package cli

import (
	"strings"
	"testing"

	"bloodhound-kube/internal/utils"
)

func TestUploadServiceRunValidation(t *testing.T) {
	service := UploadService{}
	log := utils.New("error", true)

	err := service.Run(UploadRequest{}, log)
	if err == nil || !strings.Contains(err.Error(), "token ID and token key are required") {
		t.Fatalf("expected token validation error, got %v", err)
	}

	err = service.Run(UploadRequest{TokenID: "id", TokenKey: "key"}, log)
	if err == nil || !strings.Contains(err.Error(), "provide --model-file, --queries-file, --upload-file, or --reset") {
		t.Fatalf("expected action validation error, got %v", err)
	}
}
