package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"homepage/internal/model"
	"homepage/internal/repository"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const (
	s3Provider        = "s3"
	s3PlaybackExpires = 15 * time.Minute
	s3MaxObjects      = 100
	s3MaxLyricsBytes  = 1 << 20
)

type S3Config struct {
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Prefix          string `json:"prefix"`
	ForcePathStyle  bool   `json:"force_path_style"`
	PublicBaseURL   string `json:"public_base_url"`
	AccessMode      string `json:"access_mode"`
}

type S3ConfigDTO struct {
	Endpoint       string `json:"endpoint"`
	Region         string `json:"region"`
	Bucket         string `json:"bucket"`
	AccessKeyID    string `json:"access_key_id"`
	HasSecret      bool   `json:"has_secret"`
	Prefix         string `json:"prefix"`
	ForcePathStyle bool   `json:"force_path_style"`
	PublicBaseURL  string `json:"public_base_url"`
	AccessMode     string `json:"access_mode"`
}

type S3Object struct {
	Key          string     `json:"key"`
	Size         int64      `json:"size"`
	LastModified *time.Time `json:"last_modified,omitempty"`
}

type S3ObjectList struct {
	Objects   []S3Object `json:"objects"`
	NextToken string     `json:"next_token"`
	HasMore   bool       `json:"has_more"`
}

type S3Service struct {
	repo       *repository.Repository
	httpClient *http.Client
}

func NewS3Service(repo *repository.Repository, httpClient *http.Client) *S3Service {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &S3Service{repo: repo, httpClient: httpClient}
}

func (s *S3Service) SaveConfig(config S3Config) error {
	var current S3Config
	if err := s.repo.ProviderConfig(s3Provider, &current); err != nil {
		return err
	}
	if config.SecretAccessKey == "" {
		config.SecretAccessKey = current.SecretAccessKey
	}
	normalizeS3Config(&config)
	if err := validateS3Config(config); err != nil {
		return err
	}
	return s.repo.SaveProviderConfig(s3Provider, config)
}

func (s *S3Service) ReadConfig() (S3ConfigDTO, error) {
	var config S3Config
	if err := s.repo.ProviderConfig(s3Provider, &config); err != nil {
		return S3ConfigDTO{}, err
	}
	if config.AccessMode == "" {
		config.AccessMode = "presigned"
	}
	return S3ConfigDTO{
		Endpoint:       config.Endpoint,
		Region:         config.Region,
		Bucket:         config.Bucket,
		AccessKeyID:    config.AccessKeyID,
		HasSecret:      config.SecretAccessKey != "",
		Prefix:         config.Prefix,
		ForcePathStyle: config.ForcePathStyle,
		PublicBaseURL:  config.PublicBaseURL,
		AccessMode:     config.AccessMode,
	}, nil
}

func (s *S3Service) Test(ctx context.Context) error {
	config, err := s.loadConfig()
	if err != nil {
		return err
	}
	_, err = s.client(config).HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(config.Bucket)})
	return s3Error("连接测试", err)
}

func (s *S3Service) ListObjects(ctx context.Context, prefix, continuationToken string) (S3ObjectList, error) {
	config, err := s.loadConfig()
	if err != nil {
		return S3ObjectList{}, err
	}
	prefix = strings.TrimSpace(prefix)
	if !validS3Prefix(prefix) || !validLength(continuationToken, 2048) || !plainText(continuationToken, false) {
		return S3ObjectList{}, invalid("S3 对象前缀或分页标记无效")
	}
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(config.Bucket),
		Prefix:  aws.String(config.Prefix + prefix),
		MaxKeys: aws.Int32(s3MaxObjects),
	}
	if continuationToken != "" {
		input.ContinuationToken = aws.String(continuationToken)
	}
	output, err := s.client(config).ListObjectsV2(ctx, input)
	if err != nil {
		return S3ObjectList{}, s3Error("对象列表读取", err)
	}
	result := S3ObjectList{
		Objects:   make([]S3Object, 0, len(output.Contents)),
		NextToken: aws.ToString(output.NextContinuationToken),
		HasMore:   aws.ToBool(output.IsTruncated),
	}
	for _, object := range output.Contents {
		key := aws.ToString(object.Key)
		if !allowedExtension("music", strings.ToLower(path.Ext(key))) {
			continue
		}
		result.Objects = append(result.Objects, S3Object{
			Key: key, Size: aws.ToInt64(object.Size), LastModified: object.LastModified,
		})
	}
	return result, nil
}

func (s *S3Service) Import(_ context.Context, name string, sort int, keys []string) (model.Playlist, int, error) {
	if len(keys) == 0 || len(keys) > s3MaxObjects {
		return model.Playlist{}, 0, invalid("请选择 1 至 100 个音频对象")
	}
	config, err := s.loadConfig()
	if err != nil {
		return model.Playlist{}, 0, err
	}
	playlist := model.Playlist{Name: name, Sort: sort, Enabled: true, Tracks: make([]model.MusicTrack, 0, len(keys))}
	if err := validatePlaylist(playlist); err != nil {
		return model.Playlist{}, 0, err
	}
	tracks := make([]model.MusicTrack, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for index, key := range keys {
		if !validS3ObjectKey(key) || !allowedExtension("music", strings.ToLower(path.Ext(key))) {
			return model.Playlist{}, 0, invalid("S3 Object Key 或音频扩展名无效")
		}
		if !strings.HasPrefix(key, config.Prefix) {
			return model.Playlist{}, 0, invalid("音频对象不在配置的 S3 前缀内")
		}
		if _, ok := seen[key]; ok {
			return model.Playlist{}, 0, invalid("不能重复选择同一音频对象")
		}
		seen[key] = struct{}{}
		track := model.MusicTrack{
			Title:     strings.TrimSuffix(path.Base(key), path.Ext(key)),
			Source:    s3Provider,
			ObjectKey: key,
			Sort:      index,
			Enabled:   true,
		}
		if err := validateTrack(track); err != nil {
			return model.Playlist{}, 0, err
		}
		tracks = append(tracks, track)
	}
	return s.repo.ImportPlaylist(playlist, tracks)
}

func (s *S3Service) Playback(ctx context.Context, track model.MusicTrack) (Playback, error) {
	config, err := s.loadConfig()
	if err != nil {
		return Playback{}, err
	}
	if track.Source != s3Provider || !validS3ObjectKey(track.ObjectKey) {
		return Playback{}, invalid("S3 Object Key 无效")
	}
	if config.AccessMode == "public" {
		return Playback{URL: strings.TrimRight(config.PublicBaseURL, "/") + "/" + escapeS3Key(track.ObjectKey)}, nil
	}
	request, err := s3.NewPresignClient(s.client(config)).PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(config.Bucket), Key: aws.String(track.ObjectKey),
	}, s3.WithPresignExpires(s3PlaybackExpires))
	if err != nil {
		return Playback{}, s3Error("播放地址生成", err)
	}
	expiresAt := time.Now().Add(s3PlaybackExpires)
	return Playback{URL: request.URL, ExpiresAt: &expiresAt}, nil
}

func (s *S3Service) Cover(ctx context.Context, track model.MusicTrack) (io.ReadCloser, error) {
	config, err := s.loadConfig()
	if err != nil {
		return nil, err
	}
	if track.Source != s3Provider || !validS3ObjectKey(track.ObjectKey) {
		return nil, invalid("S3 Object Key 无效")
	}
	output, err := s.client(config).GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(config.Bucket), Key: aws.String(track.ObjectKey),
	})
	if err != nil {
		return nil, s3Error("封面音频读取", err)
	}
	if aws.ToInt64(output.ContentLength) > maxCoverAudioBytes {
		_ = output.Body.Close()
		return nil, invalid("封面音频超过大小限制")
	}
	return output.Body, nil
}

func (s *S3Service) Lyrics(ctx context.Context, track model.MusicTrack) (string, error) {
	config, err := s.loadConfig()
	if err != nil {
		return "", err
	}
	if track.Source != s3Provider || !validS3ObjectKey(track.ObjectKey) {
		return "", invalid("S3 Object Key 无效")
	}
	ext := path.Ext(track.ObjectKey)
	lyricKey := strings.TrimSuffix(track.ObjectKey, ext) + ".lrc"
	output, err := s.client(config).GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(config.Bucket), Key: aws.String(lyricKey),
	})
	if isS3ObjectNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", s3Error("歌词读取", err)
	}
	defer output.Body.Close()
	if aws.ToInt64(output.ContentLength) > s3MaxLyricsBytes {
		return "", invalid("歌词文件超过大小限制")
	}
	data, err := io.ReadAll(io.LimitReader(output.Body, s3MaxLyricsBytes+1))
	if err != nil {
		return "", s3Error("歌词读取", err)
	}
	if len(data) > s3MaxLyricsBytes {
		return "", invalid("歌词文件超过大小限制")
	}
	if !utf8.Valid(data) || !plainText(string(data), true) {
		return "", invalid("歌词必须是有效的 UTF-8 文本")
	}
	return string(data), nil
}

func (s *S3Service) loadConfig() (S3Config, error) {
	var config S3Config
	if err := s.repo.ProviderConfig(s3Provider, &config); err != nil {
		return config, err
	}
	if err := validateS3Config(config); err != nil {
		return config, err
	}
	return config, nil
}

func (s *S3Service) client(config S3Config) *s3.Client {
	awsConfig := aws.Config{
		Region:       config.Region,
		Credentials:  aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, "")),
		HTTPClient:   s.httpClient,
		BaseEndpoint: aws.String(config.Endpoint),
	}
	return s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.UsePathStyle = config.ForcePathStyle
	})
}

func normalizeS3Config(config *S3Config) {
	config.Endpoint = strings.TrimSpace(config.Endpoint)
	config.Region = strings.TrimSpace(config.Region)
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.AccessKeyID = strings.TrimSpace(config.AccessKeyID)
	config.PublicBaseURL = strings.TrimRight(strings.TrimSpace(config.PublicBaseURL), "/")
	config.AccessMode = strings.TrimSpace(config.AccessMode)
	config.Prefix = strings.TrimSpace(config.Prefix)
	if config.Prefix != "" && !strings.HasSuffix(config.Prefix, "/") {
		config.Prefix += "/"
	}
}

func validateS3Config(config S3Config) error {
	if !validS3URL(config.Endpoint) || !required(config.Region, 128) || !required(config.Bucket, 128) ||
		strings.ContainsAny(config.Bucket, `/\`) || !required(config.AccessKeyID, 256) ||
		!required(config.SecretAccessKey, 512) || !validS3Prefix(config.Prefix) || !oneOf(config.AccessMode, "public", "presigned") {
		return invalid("S3 配置不完整或无效")
	}
	if config.PublicBaseURL != "" && (!validLength(config.PublicBaseURL, 2048) || !validS3URL(config.PublicBaseURL)) {
		return invalid("S3 公开访问域名无效")
	}
	if config.AccessMode == "public" && config.PublicBaseURL == "" {
		return invalid("公开 URL 模式必须配置公开访问域名")
	}
	return nil
}

func validS3URL(value string) bool {
	if !httpURL(value) {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}

func validS3Prefix(value string) bool {
	if value == "" {
		return true
	}
	if !validLength(value, 1024) || !plainText(value, false) || strings.HasPrefix(value, "/") || strings.Contains(value, `\`) {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func validS3ObjectKey(key string) bool {
	if !required(key, 1024) || strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") || strings.Contains(key, `\`) {
		return false
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func escapeS3Key(key string) string {
	segments := strings.Split(key, "/")
	for index := range segments {
		segments[index] = url.PathEscape(segments[index])
	}
	return strings.Join(segments, "/")
}

func isS3ObjectNotFound(err error) bool {
	var apiError smithy.APIError
	return errors.As(err, &apiError) && oneOf(apiError.ErrorCode(), "NoSuchKey", "NotFound")
}

func s3Error(action string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return failure(http.StatusGatewayTimeout, "s3_timeout", "S3 请求超时")
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch", "InvalidToken", "ExpiredToken":
			return failure(http.StatusBadGateway, "s3_access_denied", "S3 身份验证或权限校验失败")
		case "NoSuchBucket":
			return failure(http.StatusBadGateway, "s3_bucket_not_found", "S3 Bucket 不存在")
		case "NoSuchKey", "NotFound":
			return notFound("S3 对象或 Bucket 不存在")
		}
	}
	return failure(http.StatusBadGateway, "s3_upstream_error", "S3 "+action+"失败，请检查配置或稍后重试")
}
