package server

import (
	"cmp"
	"context"
	"time"

	"github.com/charmbracelet/log"

	pb "nagomi-receipts/internal/gen/nagomi/v1"
	"nagomi-receipts/internal/vision"
)

const (
	maxImageSize  = 5 << 20         // 5MB
	parseTimeout  = 5 * time.Minute // local models can be slow
	healthTimeout = 5 * time.Second
)

type Server struct {
	pb.UnimplementedReceiptOCRServiceServer
	model *vision.Client
	log   *log.Logger
}

func New(model *vision.Client, logger *log.Logger) *Server {
	return &Server{model: model, log: logger}
}

func (s *Server) ParseReceipt(ctx context.Context, req *pb.ParseReceiptRequest) (*pb.ParseReceiptResponse, error) {
	size := len(req.ImageData)
	switch {
	case size == 0:
		return s.fail(pb.OCRErrorCode_OCR_ERROR_INVALID_IMAGE, "empty image"), nil
	case size > maxImageSize:
		return s.fail(pb.OCRErrorCode_OCR_ERROR_INVALID_IMAGE, "image exceeds 5MB limit"), nil
	}

	contentType := cmp.Or(req.ContentType, "image/jpeg")
	s.log.Info("parsing receipt", "size_kb", size/1024, "content_type", contentType)

	ctx, cancel := context.WithTimeout(ctx, parseTimeout)
	defer cancel()

	reply, err := s.model.Ask(ctx, prompt, req.ImageData, contentType)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return s.fail(pb.OCRErrorCode_OCR_ERROR_TIMEOUT, "inference timeout"), nil
		}
		return s.fail(pb.OCRErrorCode_OCR_ERROR_MODEL_ERROR, err.Error()), nil
	}

	receipt, err := parseReceipt(reply)
	if err != nil {
		return s.fail(pb.OCRErrorCode_OCR_ERROR_PARSE_FAILED, err.Error()), nil
	}

	s.log.Info("receipt parsed",
		"merchant", deref(receipt.Merchant, "unknown"),
		"items", len(receipt.Items),
		"total", deref(receipt.Total, 0),
		"confidence", receipt.Confidence,
	)

	return &pb.ParseReceiptResponse{Success: true, Data: receipt}, nil
}

func (s *Server) Health(ctx context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()

	status := "ok"
	if err := s.model.Ping(ctx); err != nil {
		s.log.Warn("model API unreachable", "err", err)
		status = "unhealthy"
	}

	return &pb.HealthResponse{Status: status, Model: s.model.Model}, nil
}

func (s *Server) fail(code pb.OCRErrorCode, msg string) *pb.ParseReceiptResponse {
	s.log.Warn("receipt not parsed", "code", code, "err", msg)
	return &pb.ParseReceiptResponse{Error: &pb.OCRError{Code: code, Message: msg}}
}

func deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
