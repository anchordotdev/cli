package detection

import (
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestRailsDetector_Detect(t *testing.T) {
	fakeFS := fstest.MapFS{}
	for _, file := range RailsFiles {
		fakeFS[filepath.Join("rails-app", file)] = &fstest.MapFile{Data: []byte(""), Mode: 0644}
	}

	detector := RailsDetector
	detector.FileSystem = fakeFS

	match, err := detector.Detect("rails-app")

	if err != nil {
		t.Fatalf("Unexpected error during detection: %v", err)
	}

	if !match.Detected {
		t.Errorf("Expected detection result to be true, but got false")
	}

	if match.Confidence != High {
		t.Errorf("Expected confidence score to be High, but got %s", match.Confidence)
	}
}

func TestSinatraDetector_Detect(t *testing.T) {
	fakeFS := fstest.MapFS{}
	for _, file := range SinatraFiles {
		fakeFS[filepath.Join("sinatra-app", file)] = &fstest.MapFile{Data: []byte(""), Mode: 0644}
	}

	detector := SinatraDetector
	detector.FileSystem = fakeFS

	match, err := detector.Detect("sinatra-app")

	if err != nil {
		t.Fatalf("Unexpected error during detection: %v", err)
	}

	// Verify the detection result
	if !match.Detected {
		t.Errorf("Expected detection result to be true, but got false")
	}

	if match.Confidence != High {
		t.Errorf("Expected confidence score to be High, but got %s", match.Confidence)
	}
}

func TestDjangoDetector_Detect(t *testing.T) {
	fakeFS := fstest.MapFS{}
	for _, file := range DjangoFiles {
		fakeFS[filepath.Join("django-app", file)] = &fstest.MapFile{Data: []byte(""), Mode: 0644}
	}

	detector := DjangoDetector
	detector.FileSystem = fakeFS

	match, err := detector.Detect("django-app")

	if err != nil {
		t.Fatalf("Unexpected error during detection: %v", err)
	}

	// Verify the detection result
	if !match.Detected {
		t.Errorf("Expected detection result to be true, but got false")
	}

	if match.Confidence != High {
		t.Errorf("Expected confidence score to be High, but got %s", match.Confidence)
	}
}

func TestFlaskDetector_Detect(t *testing.T) {
	fakeFS := fstest.MapFS{}

	for _, file := range FlaskFiles {
		fakeFS[filepath.Join("flask-app", file)] = &fstest.MapFile{Data: []byte(""), Mode: 0644}
	}

	detector := FlaskDetector
	detector.FileSystem = fakeFS

	// Perform the detection
	match, err := detector.Detect("flask-app")

	if err != nil {
		t.Fatalf("Unexpected error during detection: %v", err)
	}

	// Verify the detection result
	if !match.Detected {
		t.Errorf("Expected detection result to be true, but got false")
	}

	if match.Confidence != High {
		t.Errorf("Expected confidence score to be High, but got %s", match.Confidence)
	}
}
