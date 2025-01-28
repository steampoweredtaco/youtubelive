package youtubelive

import (
	"bufio"
	"io"
	"maps"
	"os"
	"strings"
	"sync"
)

var config map[string]string
var configMutex sync.Mutex

// ReadFromDotFile convince function to return all values from the .env file in the
// current working directory. The values are cached after first use. The values are
// represented as key=value where the key cannot have an equal sign. Empty lines and lines
// that begin with # are ignored. This is very basic and library users probably should use
// their own configuration management.
func ReadFromDotFile() (map[string]string, error) {
	configMutex.Lock()
	defer configMutex.Unlock()
	if config != nil {
		return maps.Clone(config), nil
	}
	f, err := os.Open(".env")
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	cfg, err := ReadFromKeyValue(f)
	if err != nil {
		config = maps.Clone(cfg)
	}
	return cfg, err
}

// ReadFromKeyValue convince function to return all values from the reader. The values are
// represented as key=value where the key cannot have an equal sign. Empty lines and lines
// that begin with # are ignored.
func ReadFromKeyValue(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 1 {
			// Split will only show one result if it ends with
			// the separator for the first time, this is just a blank
			// assignment.
			result[strings.ToLower(parts[0])] = ""
		} else {
			result[strings.ToLower(parts[0])] = parts[1]
		}
	}
	return result, scanner.Err()
}

// LoadOauthCredentialsFromDotFile will return client_id, client_secret, refresh_token and additional_scopes values given the same key names, or blank/empty if not found.  additional_scopes values should be a comma seperated list of values.  This will use the cached values as if ReadFromDotFile is used.
func LoadOauthCredentialsFromDotFile() (string, string, string, []string, error) {
	env, err := ReadFromDotFile()
	if err != nil {
		return "", "", "", nil, err
	}
	clientId := env["client_id"]
	clientSecret := env["client_secret"]
	refreshToken := env["refresh_token"]
	scopesVal := env["additional_scopes"]
	splits := strings.Split(scopesVal, ",")
	scopes := make([]string, 0, 5)
	for _, scope := range splits {
		scopes = append(scopes, strings.TrimSpace(scope))
	}
	return clientId, clientSecret, refreshToken, scopes, nil
}
