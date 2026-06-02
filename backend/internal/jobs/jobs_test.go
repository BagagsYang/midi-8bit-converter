package jobs

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestInlineJobBecomesReadyWithPayloadLinks(t *testing.T) {
	now := time.Unix(100, 0)
	manager := NewManager(Config{
		DownloadTTLSeconds: 30,
		RunInline:          true,
		Now: func() time.Time {
			return now
		},
		NewID: fixedID("0123456789abcdef0123456789abcdef"),
	})

	job, err := manager.Create("lead.mid")
	if err != nil {
		t.Fatal(err)
	}
	now = time.Unix(101, 0)
	if err := manager.Start(job.ID, func() (RenderResult, error) {
		return RenderResult{DownloadName: "lead_pulse.wav", SizeBytes: 1234}, nil
	}); err != nil {
		t.Fatal(err)
	}

	readyJob, expired, ok := manager.Get(job.ID)
	if expired || !ok {
		t.Fatalf("Get() expired=%v ok=%v", expired, ok)
	}
	payload := manager.Payload(*readyJob, "/api/synthesis-jobs")
	expected := Payload{
		JobID:        job.ID,
		Status:       StatusReady,
		SourceName:   "lead.mid",
		CreatedAt:    100,
		UpdatedAt:    101,
		ExpiresAt:    131,
		DownloadName: "lead_pulse.wav",
		SizeBytes:    1234,
		DownloadURL:  "/api/synthesis-jobs/" + job.ID + "/download",
		DeleteURL:    "/api/synthesis-jobs/" + job.ID,
	}
	if !reflect.DeepEqual(payload, expected) {
		t.Fatalf("payload mismatch\nactual:   %#v\nexpected: %#v", payload, expected)
	}
}

func TestInlineJobBecomesFailedWithError(t *testing.T) {
	now := time.Unix(200, 0)
	manager := NewManager(Config{
		DownloadTTLSeconds: 60,
		RunInline:          true,
		Now: func() time.Time {
			return now
		},
		NewID: fixedID("abcdef0123456789abcdef0123456789"),
	})
	job, err := manager.Create("bad.mid")
	if err != nil {
		t.Fatal(err)
	}

	now = time.Unix(202, 0)
	if err := manager.Start(job.ID, func() (RenderResult, error) {
		return RenderResult{}, errors.New("Uploaded MIDI file is empty or incomplete.")
	}); err != nil {
		t.Fatal(err)
	}

	failedJob, expired, ok := manager.Get(job.ID)
	if expired || !ok {
		t.Fatalf("Get() expired=%v ok=%v", expired, ok)
	}
	if failedJob.Status != StatusFailed {
		t.Fatalf("status = %q", failedJob.Status)
	}
	if failedJob.Error != "Uploaded MIDI file is empty or incomplete." {
		t.Fatalf("error = %q", failedJob.Error)
	}
	if !failedJob.ExpiresAt.Equal(time.Unix(262, 0)) {
		t.Fatalf("expires_at = %s", failedJob.ExpiresAt)
	}
}

func TestAsyncQueueFullMatchesPythonMessage(t *testing.T) {
	release := make(chan struct{})
	manager := NewManager(Config{
		MaxWorkers:   1,
		MaxQueueSize: 0,
		NewID:        sequentialIDs("job00000000000000000000000000001", "job00000000000000000000000000002"),
	})

	firstJob, err := manager.Create("first.mid")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(firstJob.ID, func() (RenderResult, error) {
		<-release
		return RenderResult{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { close(release) })

	secondJob, err := manager.Create("second.mid")
	if err != nil {
		t.Fatal(err)
	}
	err = manager.Start(secondJob.ID, func() (RenderResult, error) {
		return RenderResult{}, nil
	})
	if !errors.Is(err, ErrRenderQueueFull) {
		t.Fatalf("error = %v", err)
	}
	if err == nil || err.Error() != "The render queue is full. Try again after current jobs finish." {
		t.Fatalf("queue full message = %v", err)
	}
}

func TestGetDeletesExpiredJob(t *testing.T) {
	now := time.Unix(300, 0)
	manager := NewManager(Config{
		DownloadTTLSeconds: 10,
		RunInline:          true,
		Now: func() time.Time {
			return now
		},
		NewID: fixedID("fedcba9876543210fedcba9876543210"),
	})
	job, err := manager.Create("old.mid")
	if err != nil {
		t.Fatal(err)
	}

	now = time.Unix(310, 0)
	got, expired, ok := manager.Get(job.ID)
	if got != nil || !expired || ok {
		t.Fatalf("Get() = %#v expired=%v ok=%v", got, expired, ok)
	}
	if deleted := manager.Delete(job.ID); deleted {
		t.Fatalf("expired job should already be deleted")
	}
}

func TestDeleteRemovesJob(t *testing.T) {
	manager := NewManager(Config{
		RunInline: true,
		NewID:     fixedID("00112233445566778899aabbccddeeff"),
	})
	job, err := manager.Create("delete.mid")
	if err != nil {
		t.Fatal(err)
	}
	if deleted := manager.Delete(job.ID); !deleted {
		t.Fatal("Delete() returned false")
	}
	if got, expired, ok := manager.Get(job.ID); got != nil || expired || ok {
		t.Fatalf("Get() after delete = %#v expired=%v ok=%v", got, expired, ok)
	}
}

func fixedID(id string) func() (string, error) {
	return func() (string, error) {
		return id, nil
	}
}

func sequentialIDs(ids ...string) func() (string, error) {
	index := 0
	return func() (string, error) {
		id := ids[index]
		index++
		return id, nil
	}
}
