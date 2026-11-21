package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
)

// 家长端园所二维码（#176）。
//
// 每个共用园所一张专属码：扫码落到该园，scene 语法与 #160 一致。
// 生成必须走家长端凭据：用老师端凭据会生成另一个小程序的码，家长扫码进错端。

const (
	// ParentEntryQRPage 家长端首页。必须是 parent/src/pages.json 中真实存在的页面：
	// 微信 CheckPath:false 不校验路径，写错只会静默生成死链。
	ParentEntryQRPage = "pages/index/index"
	// ParentEntryScenePrefix 园所入口 scene 前缀。
	ParentEntryScenePrefix = "kg_"
	// wechatSceneMaxLength 微信 getwxacodeunlimit 的 scene 上限（字符）。
	wechatSceneMaxLength = 32
)

var (
	ErrEntryKindergartenNotFound = errors.New("ENTRY_KINDERGARTEN_NOT_FOUND")
	ErrEntryClientUnavailable    = errors.New("ENTRY_CLIENT_UNAVAILABLE")
	ErrEntryChannelMismatch      = errors.New("ENTRY_CHANNEL_MISMATCH")
	ErrEntrySceneTooLong         = errors.New("ENTRY_SCENE_TOO_LONG")
)

// ParentClientResolver 解析当前唯一启用中的客户端（由 *ClientService 实现）。
type ParentClientResolver interface {
	FindEnabledForRole(ctx context.Context, role string) (*domain.Client, error)
}

// ParentEntryQRService 生成家长端园所入口二维码。
type ParentEntryQRService struct {
	entry   *KindergartenEntryService
	clients ParentClientResolver
	wx      WeChatAPI
}

func NewParentEntryQRService(
	entry *KindergartenEntryService,
	clients ParentClientResolver,
	wx WeChatAPI,
) *ParentEntryQRService {
	return &ParentEntryQRService{entry: entry, clients: clients, wx: wx}
}

// ParentEntryQRCode 是生成结果。PNG 以 base64 返回，沿用任务分享码的返回形状，
// 不绕过统一 JSON 包络。
type ParentEntryQRCode struct {
	PublicCode  string `json:"public_code"`
	Scene       string `json:"scene"`
	Page        string `json:"page"`
	EnvVersion  string `json:"env_version"`
	ImageBase64 string `json:"image_base64"`
}

// GenerateParentEntryQRCode 为指定园所生成家长端小程序码。
//
// 存量园没有 public_code 时先补齐再生成；同一园所重复生成始终复用同一码。
// envVersion 为空回落到部署配置，非法值显式报错（静默回落会把配置错误变成
// 「生成了另一个版本的码」）。
func (s *ParentEntryQRService) GenerateParentEntryQRCode(
	ctx context.Context,
	kindergartenID uint64,
	envVersion string,
) (*ParentEntryQRCode, error) {
	if s.entry == nil || s.entry.kg == nil || s.clients == nil || s.wx == nil {
		return nil, fmt.Errorf("%w: 家长端二维码依赖未配置", ErrEntryClientUnavailable)
	}

	kg, err := s.entry.kg.FindByID(ctx, kindergartenID)
	if err != nil {
		return nil, fmt.Errorf("读取园所失败：%w", err)
	}
	if kg == nil {
		return nil, ErrEntryKindergartenNotFound
	}

	client, err := s.clients.FindEnabledForRole(ctx, domain.ClientRoleParent)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEntryClientUnavailable, err)
	}
	// 通道矩阵是唯一判定：shared client 只能为 shared 园发码，
	// 专属/集团通道必须用各自的客户端（v1 尚不存在）。
	if !client.ServesKindergarten(kg) {
		return nil, fmt.Errorf("%w: 园所 %d（通道 %q）不由客户端 %s 服务",
			ErrEntryChannelMismatch, kg.ID, kg.AppChannel, client.AppID)
	}

	if kg.PublicCode == "" {
		if _, err := s.entry.assignPublicCode(ctx, kg.ID); err != nil {
			return nil, err
		}
		// assignPublicCode 只写库，不回填当前指针，必须重读拿到新码。
		kg, err = s.entry.kg.FindByID(ctx, kindergartenID)
		if err != nil {
			return nil, fmt.Errorf("重读园所失败：%w", err)
		}
		if kg == nil {
			return nil, ErrEntryKindergartenNotFound
		}
	}

	scene := ParentEntryScenePrefix + kg.PublicCode
	if len(scene) > wechatSceneMaxLength {
		return nil, fmt.Errorf("%w: scene %q 长度 %d 超过 %d", ErrEntrySceneTooLong, scene, len(scene), wechatSceneMaxLength)
	}

	envVersion = strings.TrimSpace(envVersion)
	if envVersion == "" {
		envVersion = s.wx.CodeEnvVersion()
	}
	switch envVersion {
	case "release", "trial", "develop":
	default:
		return nil, fmt.Errorf("%w: %q (want release|trial|develop)", auth.ErrInvalidCodeEnvVersion, envVersion)
	}

	png, err := s.wx.GetParentWXACodeUnlimit(ctx, scene, ParentEntryQRPage, envVersion)
	if err != nil {
		return nil, fmt.Errorf("生成家长端园所二维码失败：%w", err)
	}

	return &ParentEntryQRCode{
		PublicCode:  kg.PublicCode,
		Scene:       scene,
		Page:        ParentEntryQRPage,
		EnvVersion:  envVersion,
		ImageBase64: base64.StdEncoding.EncodeToString(png),
	}, nil
}
