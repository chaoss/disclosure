package detection

// Confidence represents how confident we are that a finding indicates AI involvement.
type Confidence int

const (
	ConfidenceLow    Confidence = 1 // Tool name mentioned in text
	ConfidenceMedium Confidence = 2 // Commit message pattern match
	ConfidenceHigh   Confidence = 3 // Bot email, co-author trailer, git AI ref
)

func (c Confidence) String() string {
	switch c {
	case ConfidenceLow:
		return "low"
	case ConfidenceMedium:
		return "medium"
	case ConfidenceHigh:
		return "high"
	default:
		return "unknown"
	}
}

func (c *Confidence) Increment() {
	*c = min(*c+1, ConfidenceHigh)
}

// Finding represents a single detection of AI involvement.
type Finding struct {
	Detector   string     `json:"detector"`
	Tool       string     `json:"tool"`
	Model      string     `json:"model,omitempty"`
	Version    string     `json:"version,omitempty"`
	Confidence Confidence `json:"confidence"`
	Detail     string     `json:"detail"`
}

// Input provides data for detectors to examine. Each detector reads the fields
// it cares about and ignores the rest.
type Input struct {
	CommitHash    string
	CommitEmail   string
	CommitMessage string
	Notes         string // Content from refs/notes/ai, if any
	Text          string // For text-only scans (PR body, comments)
	RepoPath      string
}

// Detector is the interface that all detection strategies implement.
type Detector interface {
	Name() string
	Detect(input Input) []Finding
}
