package service

import "github.com/Bin-hy/zhupi/internal/domain"

// BindError 在 bind 流程失败时携带已消费的流程，供 HTTP 层回跳账号页。
type BindError struct {
	Flow domain.AuthFlow
	Err  error
}

func (e *BindError) Error() string { return e.Err.Error() }

func (e *BindError) Unwrap() error { return e.Err }

func wrapBindError(flow domain.AuthFlow, err error) error {
	if err == nil || flow.Mode != domain.FlowBind {
		return err
	}
	return &BindError{Flow: flow, Err: err}
}
