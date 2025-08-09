package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"my-project/domain/model"
	"my-project/domain/repository"
	"my-project/infrastructure/logger"
)

type ShareMode string

const (
	ShareModeTrackOnly  ShareMode = "track_only"
	ShareModeServerPost ShareMode = "server_post"
)

type ShareResult struct {
	Platform      string `json:"platform"`
	Status        string `json:"status"`
	AlreadyShared bool   `json:"alreadyShared"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

type IShareUsecase interface {
	Share(ctx context.Context, videoID, userID string, platforms []string, mode ShareMode) ([]ShareResult, error)
	GetStatus(ctx context.Context, videoID, userID string) ([]*model.VideoShareRecord, error)
}

type shareUsecase struct {
	shareRepo repository.IShare
	tokenRepo repository.IOAuthToken
	allowed   map[string]struct{}
}

func NewShareUsecase(shareRepo repository.IShare, tokenRepo repository.IOAuthToken, allowed []string) IShareUsecase {
	m := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		m[strings.ToLower(a)] = struct{}{}
	}
	return &shareUsecase{shareRepo: shareRepo, tokenRepo: tokenRepo, allowed: m}
}

func (u *shareUsecase) Share(ctx context.Context, videoID, userID string, platforms []string, mode ShareMode) ([]ShareResult, error) {
	if videoID == "" || userID == "" {
		return nil, errors.New("videoID and userID required")
	}
	if len(platforms) == 0 {
		return nil, errors.New("platforms required")
	}
	norm := make([]string, 0, len(platforms))
	for _, p := range platforms {
		p = strings.ToLower(p)
		if _, ok := u.allowed[p]; !ok {
			return nil, errors.New("unsupported platform: " + p)
		}
		norm = append(norm, p)
	}
	initialStatus := "success"
	if mode == ShareModeServerPost {
		initialStatus = "pending"
	}
	records, err := u.shareRepo.UpsertTrackShares(ctx, videoID, userID, norm, initialStatus)
	if err != nil {
		return nil, err
	}
	if mode == ShareModeServerPost {
		_ = u.shareRepo.EnqueueJobs(ctx, records)
	}
	results := make([]ShareResult, 0, len(records))
	for _, r := range records {
		already := r.AttemptCount > 1 && r.Status == "success"
		msg := ""
		if r.ErrorMessage != nil {
			msg = *r.ErrorMessage
		}
		results = append(results, ShareResult{Platform: r.Platform, Status: r.Status, AlreadyShared: already, ErrorMessage: msg})
	}
	return results, nil
}

func (u *shareUsecase) GetStatus(ctx context.Context, videoID, userID string) ([]*model.VideoShareRecord, error) {
	return u.shareRepo.GetShareStatus(ctx, videoID, userID)
}

// ProcessShareJobs processes pending jobs (placeholder platform logic)
func ProcessShareJobs(ctx context.Context, shareRepo repository.IShare, tokenRepo repository.IOAuthToken, batchSize int) error {
	lg := logger.GetLogger()
	jobs, err := shareRepo.FetchPendingJobs(ctx, batchSize)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		_ = shareRepo.MarkJobRunning(ctx, job.ID)
		success := false
		var errMsg *string
		platform := strings.ToLower(job.Platform)
		switch platform {
		case "facebook":
			// Retrieve token (assumes single demo user for now). In full impl, need record join to get user/video IDs.
			// TODO: Extend share_jobs or join video_share_records to fetch user_id & video_id.
			tok, tErr := tokenRepo.GetToken(ctx, "demo-user", "facebook")
			if tErr != nil {
				m := "missing_token"
				errMsg = &m
				break
			}
			if tok.PageID == nil {
				m := "no_page_token"
				errMsg = &m
				break
			}
			// Post simple message referencing video (placeholder message - needs real video link)
			message := url.QueryEscape(fmt.Sprintf("Shared a new video at %s", time.Now().Format(time.RFC3339)))
			postURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/feed?message=%s&access_token=%s", url.PathEscape(*tok.PageID), message, url.QueryEscape(tok.AccessToken))
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, postURL, nil)
			resp, pErr := http.DefaultClient.Do(req)
			if pErr != nil {
				m := "post_request_failed"
				errMsg = &m
				break
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 200 {
				m := fmt.Sprintf("facebook_post_failed:%s", string(body))
				errMsg = &m
				break
			}
			success = true
		case "twitter":
			m := "not_implemented"
			errMsg = &m
		default:
			m := "unsupported_platform"
			errMsg = &m
		}
		_ = shareRepo.MarkJobResult(ctx, job.ID, success, errMsg)
		status := "failed"
		if success {
			status = "success"
		}
		_ = shareRepo.UpdateRecordStatus(ctx, job.RecordID, status, errMsg)
		// Audit (video_id & user_id unknown for now) - placeholder
		_ = shareRepo.CreateAudit(ctx, []*model.VideoShareAudit{{RecordID: job.RecordID, VideoID: "", Platform: platform, UserID: "", Status: status, ErrorMessage: errMsg, CreatedAt: time.Now().UTC()}})
		if errMsg != nil {
			lg.WithField("job_id", job.ID).WithField("platform", platform).WithField("error", *errMsg).Warn("share job failed")
		}
	}
	return nil
}
