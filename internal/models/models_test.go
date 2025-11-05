package models_test

import (
	"testing"

	"github.com/dragonfleas/kungfu/internal/models"
)

func TestResourceKey(t *testing.T) {
	result := models.ResourceKey("aws_instance", "web")
	expected := "aws_instance.web"

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
