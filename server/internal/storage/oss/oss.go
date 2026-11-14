package oss

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	alioss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
)

type Config struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
}

type Storage struct {
	bucket     *alioss.Bucket
	bucketName string
	endpoint   string
}

func NewStorage(cfg Config) (*Storage, error) {
	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" || cfg.BucketName == "" {
		return &Storage{bucketName: cfg.BucketName, endpoint: cfg.Endpoint}, nil
	}

	client, err := alioss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return nil, err
	}
	return &Storage{bucket: bucket, bucketName: cfg.BucketName, endpoint: cfg.Endpoint}, nil
}

func (s *Storage) PresignPut(objectKey, contentType string, ttl time.Duration) (string, error) {
	if s.bucket == nil {
		return "", fmt.Errorf("OSS not configured")
	}
	return s.bucket.SignURL(objectKey, alioss.HTTPPut, int64(ttl.Seconds()),
		alioss.ContentType(contentType),
	)
}

func (s *Storage) PresignGet(objectKey string, ttl time.Duration) (string, error) {
	return s.PresignGetWithProcess(objectKey, ttl, "")
}

// PresignGetWithProcess 签发 GET URL；process 非空时带上 OSS 图片/视频处理参数。
func (s *Storage) PresignGetWithProcess(objectKey string, ttl time.Duration, process string) (string, error) {
	if s.bucket == nil {
		return "", fmt.Errorf("OSS not configured")
	}
	opts := make([]alioss.Option, 0, 1)
	if process != "" {
		opts = append(opts, alioss.Process(process))
	}
	return s.bucket.SignURL(objectKey, alioss.HTTPGet, int64(ttl.Seconds()), opts...)
}

// Configured 表示 OSS 凭据是否完整；未配置时响应层应原样下发引用，不尝试签名。
func (s *Storage) Configured() bool {
	return s.bucket != nil
}

// ObjectRef 是从数据库引用或历史完整 URL 解析出的对象引用。
type ObjectRef struct {
	Key     string
	Process string // x-oss-process 参数（未 URL 编码）
	OSS     bool
	Signed  bool
}

// ParseRef 同时接受新格式（object key，可带 ?x-oss-process=...）与历史完整 OSS URL。
// 非 OSS 来源（例如微信头像）返回 OSS=false，调用方应原样保留。
func (s *Storage) ParseRef(ref string) ObjectRef {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ObjectRef{}
	}

	if !strings.Contains(ref, "://") {
		key, rawQuery, _ := strings.Cut(ref, "?")
		if key == "" || strings.HasPrefix(key, "/") {
			return ObjectRef{}
		}
		q, _ := url.ParseQuery(rawQuery)
		return ObjectRef{
			Key:     key,
			Process: q.Get("x-oss-process"),
			OSS:     true,
			Signed:  q.Get("Signature") != "",
		}
	}

	u, err := url.Parse(ref)
	if err != nil {
		return ObjectRef{}
	}
	host := strings.ToLower(u.Host)
	endpointHost := strings.ToLower(trimScheme(s.endpoint))
	if endpointHost == "" || (!strings.HasSuffix(host, "."+endpointHost) && host != endpointHost) {
		return ObjectRef{}
	}
	key := strings.TrimPrefix(u.Path, "/")
	if key == "" {
		return ObjectRef{}
	}
	q := u.Query()
	return ObjectRef{
		Key:     key,
		Process: q.Get("x-oss-process"),
		OSS:     true,
		Signed:  q.Get("Signature") != "",
	}
}

// SignRef 把引用签成短期 GET URL。已签名 URL、非 OSS URL 原样返回；
// thumb=true 且没有处理参数时补默认缩略图参数。
func (s *Storage) SignRef(ref string, ttl time.Duration, thumb bool) (string, error) {
	parsed := s.ParseRef(ref)
	if !parsed.OSS || parsed.Signed {
		return ref, nil
	}
	process := parsed.Process
	if thumb && process == "" {
		process = "image/resize,w_200"
	}
	return s.PresignGetWithProcess(parsed.Key, ttl, process)
}

// NormalizeRef 把客户端可能传来的完整 OSS URL（甚至已签名 URL）归一化为内部引用：
// object key，只保留 x-oss-process 参数。签名参数、过期时间等一律丢弃。
// 非 OSS URL 原样返回。
func NormalizeRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}

	if !strings.Contains(ref, "://") {
		key, rawQuery, _ := strings.Cut(ref, "?")
		if strings.HasPrefix(key, "/") {
			return ref
		}
		return key + processOnlyQuery(rawQuery)
	}

	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	host := strings.ToLower(u.Host)
	if !strings.Contains(host, "aliyuncs.com") {
		return ref
	}
	key := strings.TrimPrefix(u.Path, "/")
	if key == "" {
		return ref
	}
	return key + processOnlyQuery(u.RawQuery)
}

// ThumbRef 返回新上传文件的缩略图内部引用（key + 处理参数）。
func ThumbRef(objectKey, fileType string) string {
	switch fileType {
	case "image":
		return objectKey + "?x-oss-process=" + url.QueryEscape("image/resize,w_200")
	case "video":
		return objectKey + "?x-oss-process=" + url.QueryEscape("video/snapshot,t_1000,w_200,m_fast")
	default:
		return ""
	}
}

func processOnlyQuery(rawQuery string) string {
	q, err := url.ParseQuery(rawQuery)
	if err != nil {
		return ""
	}
	process := q.Get("x-oss-process")
	if process == "" {
		return ""
	}
	return "?x-oss-process=" + url.QueryEscape(process)
}

func (s *Storage) PutObject(objectKey string, reader io.Reader, contentType string) error {
	if s.bucket == nil {
		return fmt.Errorf("OSS not configured")
	}
	return s.bucket.PutObject(objectKey, reader,
		alioss.ContentType(contentType),
	)
}

func (s *Storage) PutFromMultipart(c *gin.Context, formKey, objectKey string) error {
	fh, err := c.FormFile(formKey)
	if err != nil {
		return err
	}
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	return s.bucket.PutObject(objectKey, src,
		alioss.ContentType(fh.Header.Get("Content-Type")),
	)
}

func (s *Storage) PublicURL(objectKey string) string {
	if s.endpoint == "" || s.bucketName == "" {
		return objectKey
	}
	return fmt.Sprintf("https://%s.%s/%s", s.bucketName, trimScheme(s.endpoint), objectKey)
}

func trimScheme(endpoint string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if len(endpoint) > len(prefix) && endpoint[:len(prefix)] == prefix {
			return endpoint[len(prefix):]
		}
	}
	return endpoint
}
