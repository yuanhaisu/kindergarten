package face

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	iai "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/iai/v20200303"

	"kindergarten-server/internal/config"
)

// TencentProvider 腾讯云人脸识别（IAI，v20200303）实现。
// 园所 → Group（kg_{id}）；学生 → Person（kg{id}_stu{sid}）。
type TencentProvider struct {
	client       *iai.Client
	qualityCtrl  uint64  // 底库入库质量档位（腾讯云 QualityControl：0=不限 1=较低 2=一般 3=较高 4=很高）
	defaultLimit uint64  // 每次搜索返回的候选数
	matchThresh  float64 // 云端分数门槛（低于此分的候选不返回——第一道降噪，我方阈值在其之上再分档）
}

const (
	// 腾讯云错误码：库已存在（幂等场景应视为成功）
	errGroupExist = "InvalidParameterValue.GroupIdAlreadyExist"
	// 人员已存在（幂等场景改为更新人脸）
	errPersonExist = "InvalidParameterValue.PersonIdAlreadyExist"
)

func NewTencentProvider(cfg config.FaceConfig) (*TencentProvider, error) {
	if cfg.SecretID == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("face: tencent provider 需要 secret_id/secret_key")
	}
	region := cfg.Region
	if region == "" {
		region = "ap-beijing"
	}
	cred := common.NewCredential(cfg.SecretID, cfg.SecretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "iai.tencentcloudapi.com"
	cpf.HttpProfile.ReqTimeout = 15
	client, err := iai.NewClient(cred, region, cpf)
	if err != nil {
		return nil, fmt.Errorf("face: new iai client: %w", err)
	}
	// 云端门槛取 suggest_low：低于"低置信"的候选我方本来就不落建议，让云端先砍掉
	thresh := float64(cfg.SuggestLow)
	if thresh <= 0 {
		thresh = 70
	}
	// 质量档位 2（一般）：底库照多为家长随手拍，过高档位会大量拒绝入库
	return &TencentProvider{client: client, qualityCtrl: 2, defaultLimit: 5, matchThresh: thresh}, nil
}

func (p *TencentProvider) Name() string { return "tencent" }

// ensureGroup 幂等创建园所人员库。
func (p *TencentProvider) ensureGroup(gid string) error {
	req := iai.NewCreateGroupRequest()
	req.GroupId = common.StringPtr(gid)
	req.GroupName = common.StringPtr(gid)
	_, err := p.client.CreateGroup(req)
	if err != nil {
		if isSDKCode(err, errGroupExist) {
			return nil
		}
		return err
	}
	return nil
}

// EnsurePerson 幂等注册：库不存在先建库；人员不存在则创建，已存在则补加人脸。
func (p *TencentProvider) EnsurePerson(ctx context.Context, kgID, studentID uint64, images [][]byte) (string, error) {
	if len(images) == 0 {
		return "", fmt.Errorf("face: EnsurePerson 需要至少一张人脸照片")
	}
	gid := groupID(kgID)
	pid := personID(kgID, studentID)
	if err := p.ensureGroup(gid); err != nil {
		return "", fmt.Errorf("face: ensure group %s: %w", gid, err)
	}

	// 先尝试创建人员（携带第一张照片）
	addFrom := 1 // 默认首张已随 CreatePerson 入库
	createReq := iai.NewCreatePersonRequest()
	createReq.GroupId = common.StringPtr(gid)
	createReq.PersonId = common.StringPtr(pid)
	createReq.PersonName = common.StringPtr(pid)
	createReq.Image = common.StringPtr(base64Std(images[0]))
	createReq.QualityControl = common.Uint64Ptr(p.qualityCtrl)
	if _, err := p.client.CreatePerson(createReq); err != nil {
		if !isSDKCode(err, errPersonExist) {
			return "", fmt.Errorf("face: CreatePerson %s: %w", pid, err)
		}
		// 人员已存在：images[0] 并未入库，需一并补加（否则单图追加是静默空操作）
		addFrom = 0
	}
	// 其余照片补加为额外人脸样本（已有样本时腾讯云按图去重）
	for _, img := range images[addFrom:] {
		req := iai.NewCreateFaceRequest()
		req.PersonId = common.StringPtr(pid)
		req.Images = []*string{common.StringPtr(base64Std(img))}
		req.QualityControl = common.Uint64Ptr(p.qualityCtrl)
		if _, err := p.client.CreateFace(req); err != nil {
			return "", fmt.Errorf("face: CreateFace %s: %w", pid, err)
		}
	}
	return pid, nil
}

// RemovePerson 幂等移除（人员不存在视为成功）。
func (p *TencentProvider) RemovePerson(ctx context.Context, kgID, studentID uint64) error {
	req := iai.NewDeletePersonRequest()
	req.PersonId = common.StringPtr(personID(kgID, studentID))
	if _, err := p.client.DeletePerson(req); err != nil {
		if strings.Contains(sdkCode(err), "PersonNotFound") || strings.Contains(sdkCode(err), "ResourceNotFound") {
			return nil
		}
		return err
	}
	return nil
}

// Search 单张照片 1:N 搜索，限定该园底库（SearchFacesReturnsByGroup 的 Groups 限定）。
func (p *TencentProvider) Search(ctx context.Context, kgID uint64, image []byte) (SearchResult, error) {
	req := iai.NewSearchFacesReturnsByGroupRequest()
	req.GroupIds = []*string{common.StringPtr(groupID(kgID))}
	req.Image = common.StringPtr(base64Std(image))
	limit := uint64(p.defaultLimit)
	req.MaxPersonNumPerGroup = &limit
	// 一张合影可有多个孩子：识别全部人脸（腾讯云上限 10）
	req.MaxFaceNum = common.Uint64Ptr(10)
	// 人脸最小尺寸：官方建议 80（低于 34 影响精度）；班级合影按 40 起步兼顾远脸
	req.MinFaceSize = common.Uint64Ptr(40)
	// 云端分数门槛：低于"低置信"的候选我方本就不落建议，让云端先砍（降噪省流量）
	req.FaceMatchThreshold = common.Float64Ptr(p.matchThresh)
	resp, err := p.client.SearchFacesReturnsByGroup(req)
	if err != nil {
		// 图中无人脸：非错误，等价于「无命中」
		if isSDKCode(err, "InvalidParameterValue.NoFaceInPhoto") {
			return SearchResult{}, nil
		}
		return SearchResult{}, err
	}

	result := SearchResult{}
	if resp.Response.FaceNum != nil {
		result.FaceNum = int(*resp.Response.FaceNum)
	}
	for _, r := range resp.Response.ResultsReturnsByGroup {
		for _, gc := range r.GroupCandidates {
			for _, cand := range gc.Candidates {
				if cand.PersonId == nil || cand.Score == nil {
					continue
				}
				sid, ok := parseStudentID(*cand.PersonId)
				if !ok {
					continue // 非本系统的 PersonId，防御性跳过
				}
				result.Matches = append(result.Matches, Match{StudentID: sid, Score: *cand.Score})
			}
		}
	}
	return result, nil
}

// parseStudentID 从 kg{id}_stu{sid} 解析学生 id；不匹配返回 false。
func parseStudentID(pid string) (uint64, bool) {
	i := strings.Index(pid, "_stu")
	if i < 0 {
		return 0, false
	}
	sid, err := strconv.ParseUint(pid[i+len("_stu"):], 10, 64)
	if err != nil {
		return 0, false
	}
	return sid, true
}

// base64Std 标准 base64（腾讯云 Image 参数要求）。
func base64Std(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func sdkCode(err error) string {
	if e, ok := err.(*tcerr.TencentCloudSDKError); ok {
		return e.GetCode()
	}
	return ""
}

func isSDKCode(err error, code string) bool {
	return sdkCode(err) == code
}
