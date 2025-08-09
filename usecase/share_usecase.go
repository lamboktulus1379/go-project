package usecase

import (
    "context"
    "errors"
    "strings"
    "time"
    "my-project/domain/model"
    "my-project/domain/repository"
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
    for _, a := range allowed { m[strings.ToLower(a)] = struct{}{} }
    return &shareUsecase{shareRepo: shareRepo, tokenRepo: tokenRepo, allowed: m}
}

func (u *shareUsecase) Share(ctx context.Context, videoID, userID string, platforms []string, mode ShareMode) ([]ShareResult, error) {
    if videoID == "" || userID == "" { return nil, errors.New("videoID and userID required") }
    if len(platforms)==0 { return nil, errors.New("platforms required") }
    norm := make([]string,0,len(platforms))
    for _, p := range platforms { p = strings.ToLower(p); if _,ok:=u.allowed[p]; !ok { return nil, errors.New("unsupported platform: "+p) }; norm = append(norm,p) }
    initialStatus := "success"
    if mode == ShareModeServerPost { initialStatus = "pending" }
    records, err := u.shareRepo.UpsertTrackShares(ctx, videoID, userID, norm, initialStatus)
    if err != nil { return nil, err }
    if mode == ShareModeServerPost {
        _ = u.shareRepo.EnqueueJobs(ctx, records)
    }
    results := make([]ShareResult, 0, len(records))
    for _, r := range records {
        already := r.AttemptCount > 1 && r.Status == "success"
        msg := ""
        if r.ErrorMessage != nil { msg = *r.ErrorMessage }
        results = append(results, ShareResult{Platform: r.Platform, Status: r.Status, AlreadyShared: already, ErrorMessage: msg})
    }
    return results, nil
}

func (u *shareUsecase) GetStatus(ctx context.Context, videoID, userID string) ([]*model.VideoShareRecord, error) {
    return u.shareRepo.GetShareStatus(ctx, videoID, userID)
}

// ProcessShareJobs processes pending jobs (placeholder platform logic)
func ProcessShareJobs(ctx context.Context, shareRepo repository.IShare, tokenRepo repository.IOAuthToken, batchSize int) error {
    jobs, err := shareRepo.FetchPendingJobs(ctx, batchSize)
    if err != nil { return err }
    for _, job := range jobs {
        _ = shareRepo.MarkJobRunning(ctx, job.ID)
        success := false
        var errMsg *string
        switch job.Platform {
        case "twitter":
            m := "not implemented"
            errMsg = &m
        case "facebook":
            m := "not implemented"
            errMsg = &m
        default:
            m := "server_post not supported"
            errMsg = &m
        }
        _ = shareRepo.MarkJobResult(ctx, job.ID, success, errMsg)
        status := "failed"
        if success { status = "success" }
        _ = shareRepo.UpdateRecordStatus(ctx, job.RecordID, status, errMsg)
        _ = shareRepo.CreateAudit(ctx, []*model.VideoShareAudit{ { RecordID: job.RecordID, VideoID: "", Platform: job.Platform, UserID: "", Status: status, ErrorMessage: errMsg, CreatedAt: time.Now().UTC() } })
    }
    return nil
}
