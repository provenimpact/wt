package shell

import "fmt"

const bashZshFunc = `wt() {
  local output
  output=$(command wt "$@")
  local exit_code=$?
  if [[ "$output" == __wt_cd:* ]]; then
    cd "${output#__wt_cd:}"
  elif [[ -n "$output" ]]; then
    echo "$output"
  fi
  return $exit_code
}
`

const fishFunc = `function wt
  set -l output (command wt $argv)
  set -l exit_code $status
  if string match -q '__wt_cd:*' $output
    cd (string replace '__wt_cd:' '' $output)
  else if test -n "$output"
    echo $output
  end
  return $exit_code
end
`

// Generate returns the shell function code for the given shell name.
func Generate(shellName string) (string, error) {
	switch shellName {
	case "bash", "zsh":
		return bashZshFunc, nil
	case "fish":
		return fishFunc, nil
	default:
		return "", fmt.Errorf("unsupported shell %q; supported: bash, zsh, fish", shellName)
	}
}
