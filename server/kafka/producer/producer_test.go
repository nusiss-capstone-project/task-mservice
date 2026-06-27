package producer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nusiss-capstone-project/task-mservice/server/config"
)

func TestGetKafkaProducerSingleton(t *testing.T) {
	p1 := GetKafkaProducer()
	p2 := GetKafkaProducer()
	if p1 != p2 {
		t.Fatal("expected singleton instance")
	}
}

func TestNopKafkaProducer(t *testing.T) {
	p := nopKafkaProducer{}
	err := p.Publish(context.Background(), "test.topic", []byte("key"), []byte("value"))
	if err != nil {
		t.Fatalf("nop producer error = %v", err)
	}
}

func TestValidateProducerConfig(t *testing.T) {
	if err := validateConfig(nil); err == nil {
		t.Fatal("expected error for nil config")
	}
	if err := validateConfig(&config.KafkaConfig{}); err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestBuildProducerDisabled(t *testing.T) {
	p := buildProducer(&config.KafkaConfig{Enabled: false})
	if _, ok := p.(nopKafkaProducer); !ok {
		t.Fatalf("expected nop producer, got %T", p)
	}
}

func TestGetTaskCompleteProducerSingleton(t *testing.T) {
	p1 := GetTaskCompleteProducer()
	p2 := GetTaskCompleteProducer()
	if p1 != p2 {
		t.Fatal("expected singleton instance")
	}
}

func TestValidateTaskCompletedInput(t *testing.T) {
	if err := validateTaskCompletedInput(0, 1, TaskCompletionStatusCompleted); err == nil {
		t.Fatal("expected error for invalid task_id")
	}
	if err := validateTaskCompletedInput(1, 0, TaskCompletionStatusCompleted); err == nil {
		t.Fatal("expected error for invalid user_id")
	}
	if err := validateTaskCompletedInput(1, 1, TaskCompletionStatus("unknown")); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestTaskCompletedEventJSON(t *testing.T) {
	payload, err := json.Marshal(TaskCompletedEvent{
		TaskID: 10,
		UserID: 20,
		Status: TaskCompletionStatusExpired,
	})
	if err != nil {
		t.Fatalf("marshal error = %v", err)
	}
	expected := `{"task_id":10,"user_id":20,"status":"expired"}`
	if string(payload) != expected {
		t.Fatalf("payload = %s, want %s", payload, expected)
	}
}

func TestPublishRequiresTopic(t *testing.T) {
	p := &kafkaProducerImpl{}
	err := p.Publish(context.Background(), "", []byte("k"), []byte("v"))
	if err == nil {
		t.Fatal("expected error for empty topic")
	}
}
