package runtime

import "os/exec"

type DoctorResult struct {
	Name    string
	Version string
	Err     error
}

var doctorCommands = []struct {
	name string
	cmd  []string
}{
	{"git", []string{"git", "--version"}},
	{"curl", []string{"curl", "--version"}},
	{"sops", []string{"sops", "--version"}},
	{"age", []string{"age", "--version"}},
	{"ansible", []string{"ansible", "--version"}},
	{"lockctl", []string{"lockctl", "version"}},
}

func RunDoctor() []DoctorResult {
	var results []DoctorResult
	for _, entry := range doctorCommands {
		res := DoctorResult{Name: entry.name}
		cmd := exec.Command(entry.cmd[0], entry.cmd[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			res.Err = err
		} else {
			res.Version = parseVersionLine(string(output))
		}
		results = append(results, res)
	}
	return results
}

func MissingTools(results []DoctorResult) []string {
	var missing []string
	for _, r := range results {
		if r.Err != nil {
			missing = append(missing, r.Name)
		}
	}
	return missing
}

func parseVersionLine(out string) string {
	line := out
	for i, ch := range out {
		if ch == '\n' || ch == '\r' {
			line = out[:i]
			break
		}
	}
	return line
}
