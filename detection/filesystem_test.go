package detection

import (
	"os"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/anchordotdev/cli/anchorcli"
)

func TestFileDetector_Detect(t *testing.T) {
	emptyFile := &fstest.MapFile{Data: []byte(""), Mode: 0644}

	testCases := []struct {
		name        string
		detector    FileDetector
		directory   string
		expected    Match
		expectError bool
	}{
		{
			name: "Mock Exact Match",
			detector: FileDetector{
				Title: "Test Detector",
				Paths: []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
				FileSystem: fstest.MapFS{
					"app/1.txt": emptyFile,
					"app/2.txt": emptyFile,
					"app/3.txt": emptyFile,
					"app/4.txt": emptyFile,
					"app/5.txt": emptyFile,
				},
				AnchorCategory: anchorcli.CategoryCustom,
			},
			directory:   "app/",
			expected:    Match{Detected: true, Confidence: High, FollowUpDetectors: nil, AnchorCategory: anchorcli.CategoryCustom},
			expectError: false,
		},
		{
			name: "Mock Strong Match",
			detector: FileDetector{
				Title: "Test Detector",
				Paths: []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
				FileSystem: fstest.MapFS{
					"app/1.txt": emptyFile,
					"app/2.txt": emptyFile,
					"app/3.txt": emptyFile,
				},
			},
			directory:   "app/",
			expected:    Match{Detected: true, Confidence: Medium, FollowUpDetectors: nil, AnchorCategory: anchorcli.CategoryCustom},
			expectError: false,
		},
		{
			name: "Mock Possible Match",
			detector: FileDetector{
				Title: "Test Detector",
				Paths: []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
				FileSystem: fstest.MapFS{
					"app/1.txt": emptyFile,
					"app/2.txt": emptyFile,
				},
			},
			directory:   "app/",
			expected:    Match{Detected: true, Confidence: Low, FollowUpDetectors: nil, AnchorCategory: anchorcli.CategoryCustom},
			expectError: false,
		},
		{
			name: "Mock None Match",
			detector: FileDetector{
				Title:      "Test Detector",
				Paths:      []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
				FileSystem: fstest.MapFS{},
			},
			directory:   "app/",
			expected:    Match{Detected: false, Confidence: None, FollowUpDetectors: nil, AnchorCategory: anchorcli.CategoryCustom},
			expectError: false,
		},
		{
			name: "Missing RequiredFiles forces match to Low",
			detector: FileDetector{
				Title:         "Test Detector",
				Paths:         []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"},
				RequiredFiles: []string{"1.txt"},
				FileSystem: fstest.MapFS{
					"app/2.txt": emptyFile,
					"app/3.txt": emptyFile,
					"app/4.txt": emptyFile,
					"app/5.txt": emptyFile,
				},
			},
			directory:   "app/",
			expected:    Match{Detected: true, Confidence: Low, FollowUpDetectors: nil, AnchorCategory: anchorcli.CategoryCustom, MissingRequiredFiles: []string{"1.txt"}},
			expectError: false,
		},
		{
			name: "Missing Required Files never forces None match to Low",
			detector: FileDetector{
				Title:         "Test Detector",
				Paths:         []string{"20.txt", "40.txt", "60.txt", "80.txt", "100.txt"},
				RequiredFiles: []string{"1.txt"},
				FileSystem: fstest.MapFS{
					"app/20.txt": emptyFile,
				},
			},
			directory:   "app/",
			expected:    Match{Detected: true, Confidence: None, FollowUpDetectors: nil, AnchorCategory: anchorcli.CategoryCustom, MissingRequiredFiles: []string{"1.txt"}},
			expectError: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			match, err := testCase.detector.Detect(testCase.directory)

			if testCase.expectError && err == nil {
				t.Errorf("Expected an error, but got none")
			}

			if !testCase.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if match.Detected != testCase.expected.Detected {
				t.Errorf("Expected detection result %t, but got %t", testCase.expected.Detected, match.Detected)
			}

			if match.Confidence != testCase.expected.Confidence {
				t.Errorf("Expected confidence score %s, but got %s", testCase.expected.Confidence, match.Confidence)
			}

			if len(match.FollowUpDetectors) != len(testCase.expected.FollowUpDetectors) {
				t.Errorf("Expected %d follow-up detectors, but got %d", len(testCase.expected.FollowUpDetectors), len(match.FollowUpDetectors))
			}

			if match.AnchorCategory != testCase.expected.AnchorCategory {
				t.Errorf("Expected AnchorCategory %s, but got %s", testCase.expected.AnchorCategory, match.AnchorCategory)
			}

			if testCase.expected.MissingRequiredFiles != nil && slices.Compare(match.MissingRequiredFiles, testCase.expected.MissingRequiredFiles) != 0 {
				t.Errorf("Expected missing required files %v, but got %v", testCase.expected.MissingRequiredFiles, match.MissingRequiredFiles)
			}
		})
	}
}

func TestFileDetector_DetectWithoutMockFS(t *testing.T) {
	// Specify the name of an existing directory that contains the expected files
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unexpected Getwd error: %v", err)
	}

	files, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("Unexpected ReadDir error: %v", err)
	}

	// Choose a random file from working directory
	expectedFiles := []string{files[0].Name()}

	// Create the FileDetector with the expected file paths
	// Instantiating one without a FileSystem will use the actual OS File System
	detector := FileDetector{
		Title: "Test Detector",
		Paths: expectedFiles,
	}

	// Perform the detection
	match, err := detector.Detect(directory)

	if err != nil {
		t.Fatalf("Unexpected error during detection: %v", err)
	}

	// Verify the detection result
	if !match.Detected {
		t.Errorf("Expected detection result to be true, but got false")
	}

	if match.Confidence != High {
		t.Errorf("Expected confidence to be High, but got %s", match.Confidence)
	}
}

func TestFileDetector_GetTitle(t *testing.T) {
	detector := FileDetector{
		Title: "Test Detector",
		Paths: []string{
			"file1.txt",
			"file2.txt",
		},
	}

	expectedTitle := "Test Detector"
	actualTitle := detector.GetTitle()

	if actualTitle != expectedTitle {
		t.Errorf("Expected detector name %s, but got %s", expectedTitle, actualTitle)
	}
}

func TestFileDetector_FollowUp(t *testing.T) {
	// Create a FileDetector with some follow-up detectors
	detector1 := FileDetector{Title: "Detector 1", Paths: []string{"file1.txt"}}
	detector2 := FileDetector{Title: "Detector 2", Paths: []string{"file2.txt"}}
	detector3 := FileDetector{Title: "Detector 3", Paths: []string{"file3.txt"}}

	parentDetector := FileDetector{
		Title:             "Parent Detector",
		Paths:             []string{"parent.txt"},
		FollowUpDetectors: []Detector{&detector1, &detector2, &detector3},
	}

	// Get the follow-up detectors
	followUpDetectors := parentDetector.FollowUp()

	// Verify the expected follow-up detectors
	expectedFollowUpDetectors := []Detector{&detector1, &detector2, &detector3}
	if len(followUpDetectors) != len(expectedFollowUpDetectors) {
		t.Errorf("Expected %d follow-up detectors, but got %d", len(expectedFollowUpDetectors), len(followUpDetectors))
		return
	}

	for i, expectedDetector := range expectedFollowUpDetectors {
		actualDetector := followUpDetectors[i]
		if actualDetector.GetTitle() != expectedDetector.GetTitle() {
			t.Errorf("Expected follow-up detector name %s, but got %s", expectedDetector.GetTitle(), actualDetector.GetTitle())
		}
	}
}
