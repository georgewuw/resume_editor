package resume

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"resume-editor/internal/apperr"
)

const (
	defaultWebDAVServerURL = "https://dav.jianguoyun.com/dav/"
	defaultWebDAVPath      = "resume-editor"
	maxSyncedDocuments     = 10000
)

type WebDAVSettings struct {
	ServerURL    string `json:"serverURL"`
	Username     string `json:"username"`
	RemotePath   string `json:"remotePath"`
	HasPassword  bool   `json:"hasPassword"`
	LastSyncedAt string `json:"lastSyncedAt"`
}

type WebDAVSettingsInput struct {
	ServerURL     string `json:"serverURL"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	RemotePath    string `json:"remotePath"`
	ClearPassword bool   `json:"clearPassword"`
}

type WebDAVSyncResult struct {
	Uploaded     int      `json:"uploaded"`
	Downloaded   int      `json:"downloaded"`
	Deleted      int      `json:"deleted"`
	Conflicts    []string `json:"conflicts"`
	LastSyncedAt string   `json:"lastSyncedAt"`
}

type webDAVConfig struct {
	ServerURL  string `json:"serverURL"`
	Username   string `json:"username"`
	Password   string `json:"-"`
	RemotePath string `json:"remotePath"`
}

type webDAVFileState struct {
	Hash string `json:"hash"`
	ETag string `json:"etag,omitempty"`
}

type webDAVSyncState struct {
	ConfigKey    string                     `json:"configKey"`
	Files        map[string]webDAVFileState `json:"files"`
	LastSyncedAt string                     `json:"lastSyncedAt"`
}

type webDAVSyncAction struct {
	kind     string
	filename string
	data     []byte
	hash     string
	etag     string
}

func (r *Resume) webDAVConfigPath() string {
	return filepath.Join(filepath.Dir(r.paths.ContentFile), "webdav-settings.json")
}

func (r *Resume) webDAVStatePath() string {
	return filepath.Join(filepath.Dir(r.paths.ContentFile), "webdav-sync-state.json")
}

func (r *Resume) readWebDAVConfig() (webDAVConfig, error) {
	config := webDAVConfig{ServerURL: defaultWebDAVServerURL, RemotePath: defaultWebDAVPath}
	data, err := os.ReadFile(r.webDAVConfigPath())
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return webDAVConfig{}, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return webDAVConfig{}, err
	}
	if config.ServerURL == "" {
		config.ServerURL = defaultWebDAVServerURL
	}
	if config.RemotePath == "" {
		config.RemotePath = defaultWebDAVPath
	}
	password, err := r.webDAVCredentials.Get(webDAVCredentialService, webDAVCredentialAccount(config))
	if err != nil && !errors.Is(err, errWebDAVCredentialNotFound) {
		return webDAVConfig{}, err
	}
	config.Password = password
	return config, nil
}

// GetWebDAVSettings never returns the stored password to the frontend.
func (r *Resume) GetWebDAVSettings() (WebDAVSettings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	config, err := r.readWebDAVConfig()
	if err != nil {
		return WebDAVSettings{}, r.failure("webdav_settings_read", apperr.StorageRead, err)
	}
	state, err := r.readWebDAVState(webDAVConfigKey(config))
	if err != nil {
		return WebDAVSettings{}, r.failure("webdav_state_read", apperr.StorageRead, err)
	}
	return publicWebDAVSettings(config, state.LastSyncedAt), nil
}

func publicWebDAVSettings(config webDAVConfig, lastSyncedAt string) WebDAVSettings {
	return WebDAVSettings{
		ServerURL: config.ServerURL, Username: config.Username, RemotePath: config.RemotePath,
		HasPassword: config.Password != "", LastSyncedAt: lastSyncedAt,
	}
}

// SaveWebDAVSettings preserves a saved password when the password field is blank.
func (r *Resume) SaveWebDAVSettings(input WebDAVSettingsInput) (WebDAVSettings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	config, err := r.webDAVConfigFromInput(input)
	if err != nil {
		return WebDAVSettings{}, r.failure("webdav_settings_validate", apperr.WebDAVConfig, err)
	}
	if err := r.writeWebDAVConfig(config); err != nil {
		return WebDAVSettings{}, r.failure("webdav_settings_write", apperr.StorageWrite, err)
	}
	state, err := r.readWebDAVState(webDAVConfigKey(config))
	if err != nil {
		return WebDAVSettings{}, r.failure("webdav_state_read", apperr.StorageRead, err)
	}
	return publicWebDAVSettings(config, state.LastSyncedAt), nil
}

func (r *Resume) writeWebDAVConfig(config webDAVConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.webDAVConfigPath()), 0o700); err != nil {
		return err
	}

	account := webDAVCredentialAccount(config)
	previousValue, hadPreviousValue, err := r.readWebDAVCredential(account)
	if err != nil {
		return err
	}
	if config.Password == "" {
		err = r.deleteWebDAVCredential(account)
	} else {
		err = r.webDAVCredentials.Set(webDAVCredentialService, account, config.Password)
	}
	if err != nil {
		return err
	}
	if err := writeDocumentFile(r.webDAVConfigPath(), data); err != nil {
		if rollbackErr := r.restoreWebDAVCredential(account, hadPreviousValue, previousValue); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}
	return nil
}

func (r *Resume) readWebDAVCredential(account string) (string, bool, error) {
	password, err := r.webDAVCredentials.Get(webDAVCredentialService, account)
	if errors.Is(err, errWebDAVCredentialNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return password, true, nil
}

func (r *Resume) restoreWebDAVCredential(account string, existed bool, password string) error {
	if !existed {
		return r.deleteWebDAVCredential(account)
	}
	return r.webDAVCredentials.Set(webDAVCredentialService, account, password)
}

func (r *Resume) deleteWebDAVCredential(account string) error {
	err := r.webDAVCredentials.Delete(webDAVCredentialService, account)
	if errors.Is(err, errWebDAVCredentialNotFound) {
		return nil
	}
	return err
}

func (r *Resume) webDAVConfigFromInput(input WebDAVSettingsInput) (webDAVConfig, error) {
	config := webDAVConfig{
		ServerURL:  strings.TrimSpace(input.ServerURL),
		Username:   strings.TrimSpace(input.Username),
		RemotePath: strings.TrimSpace(input.RemotePath),
	}
	if config.ServerURL == "" {
		config.ServerURL = defaultWebDAVServerURL
	}
	if config.RemotePath == "" {
		config.RemotePath = defaultWebDAVPath
	}
	if err := validateWebDAVServerURL(config.ServerURL); err != nil {
		return webDAVConfig{}, err
	}
	if config.Username == "" || strings.ContainsAny(config.Username, "\r\n") {
		return webDAVConfig{}, errors.New("missing or invalid WebDAV credentials")
	}
	password, _, err := r.readWebDAVCredential(webDAVCredentialAccount(config))
	if err != nil {
		return webDAVConfig{}, err
	}
	config.Password = password
	if input.ClearPassword {
		config.Password = ""
	} else if input.Password != "" {
		config.Password = input.Password
	}
	if _, err := newWebDAVClient(config.ServerURL, config.Username, config.Password); err != nil {
		return webDAVConfig{}, err
	}
	if strings.ContainsAny(config.Password, "\r\n") {
		return webDAVConfig{}, errors.New("missing or invalid WebDAV credentials")
	}
	remotePath, err := normalizeWebDAVPath(config.RemotePath)
	if err != nil {
		return webDAVConfig{}, err
	}
	config.RemotePath = remotePath
	return config, nil
}

func normalizeWebDAVPath(value string) (string, error) {
	value = strings.TrimSpace(strings.Trim(value, "/"))
	if value == "" {
		return "", nil
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, "\\?#") {
			return "", errors.New("invalid WebDAV directory path")
		}
	}
	return strings.Join(parts, "/"), nil
}

func validateWebDAVServerURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("invalid WebDAV server URL")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme != "http" {
		return errors.New("WebDAV server URL must use HTTPS")
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
		return errors.New("WebDAV server URL must use HTTPS")
	}
	return nil
}

func requireWebDAVCredentials(config webDAVConfig) error {
	if err := validateWebDAVServerURL(config.ServerURL); err != nil {
		return err
	}
	if config.Username == "" || config.Password == "" {
		return errors.New("WebDAV account and application password are required")
	}
	return nil
}

func (r *Resume) webDAVConfigKey() (webDAVConfig, string, error) {
	config, err := r.readWebDAVConfig()
	if err != nil {
		return webDAVConfig{}, "", err
	}
	if err := requireWebDAVCredentials(config); err != nil {
		return webDAVConfig{}, "", err
	}
	return config, webDAVConfigKey(config), nil
}

func webDAVConfigKey(config webDAVConfig) string {
	key := strings.Join([]string{config.ServerURL, config.Username, config.RemotePath}, "\x00")
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:])
}

func (r *Resume) readWebDAVState(configKey string) (webDAVSyncState, error) {
	state := webDAVSyncState{ConfigKey: configKey, Files: map[string]webDAVFileState{}}
	data, err := os.ReadFile(r.webDAVStatePath())
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return webDAVSyncState{}, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return webDAVSyncState{}, err
	}
	if state.ConfigKey != configKey {
		return webDAVSyncState{ConfigKey: configKey, Files: map[string]webDAVFileState{}}, nil
	}
	if state.Files == nil {
		state.Files = map[string]webDAVFileState{}
	}
	return state, nil
}

func (r *Resume) writeWebDAVState(state webDAVSyncState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.webDAVStatePath()), 0o700); err != nil {
		return err
	}
	return writeDocumentFile(r.webDAVStatePath(), data)
}

// TestWebDAVConnection verifies credentials without changing remote files.
func (r *Resume) TestWebDAVConnection(input WebDAVSettingsInput) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	config, err := r.webDAVConfigFromInput(input)
	if err != nil {
		return "", r.failure("webdav_connection", apperr.WebDAVConfig, err)
	}
	if err := requireWebDAVCredentials(config); err != nil {
		return "", r.failure("webdav_connection", apperr.WebDAVConfig, err)
	}
	client, err := newWebDAVClient(config.ServerURL, config.Username, config.Password)
	if err != nil {
		return "", r.failure("webdav_connection", apperr.WebDAVConfig, err)
	}
	err = client.probe(config.RemotePath)
	if err == nil {
		return "连接成功", nil
	}
	if isWebDAVStatus(err, http.StatusNotFound) && config.RemotePath != "" {
		if rootErr := client.probe(""); rootErr == nil {
			return "连接成功，云端目录将在首次同步时创建", nil
		}
	}
	return "", r.failure("webdav_connection", apperr.WebDAVConnect, err)
}

// SyncWebDAV manually synchronizes only the application's managed Markdown files.
func (r *Resume) SyncWebDAV() (WebDAVSyncResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	config, key, err := r.webDAVConfigKey()
	if err != nil {
		return WebDAVSyncResult{}, r.failure("webdav_sync_config", apperr.WebDAVConfig, err)
	}
	client, err := newWebDAVClient(config.ServerURL, config.Username, config.Password)
	if err != nil {
		return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVConfig, err)
	}
	state, err := r.readWebDAVState(key)
	if err != nil {
		return WebDAVSyncResult{}, r.failure("webdav_state_read", apperr.StorageRead, err)
	}
	remoteEntries, err := client.readDir(config.RemotePath)
	if isWebDAVStatus(err, http.StatusNotFound) {
		if len(state.Files) != 0 {
			return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, errors.New("configured remote directory was removed"))
		}
		if err := client.makeDirs(config.RemotePath); err != nil {
			return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, err)
		}
		remoteEntries, err = client.readDir(config.RemotePath)
	}
	if err != nil {
		return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, err)
	}
	if len(remoteEntries) > maxSyncedDocuments {
		return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, errors.New("too many managed Markdown files in the remote directory"))
	}
	localFiles, err := r.readLocalSyncFiles()
	if err != nil {
		return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, err)
	}
	remoteFiles := make(map[string]webDAVEntry, len(remoteEntries))
	remoteData := make(map[string][]byte, len(remoteEntries))
	remoteHashes := make(map[string]string, len(remoteEntries))
	for _, entry := range remoteEntries {
		remoteFiles[entry.name] = entry
		previous := state.Files[entry.name]
		if entry.etag != "" && entry.etag == previous.ETag && previous.Hash != "" {
			remoteHashes[entry.name] = previous.Hash
			continue
		}
		data, err := client.read(config.RemotePath, entry.name)
		if err != nil {
			return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, err)
		}
		remoteData[entry.name] = data
		remoteHashes[entry.name] = markdownHash(data)
	}

	result := WebDAVSyncResult{Conflicts: []string{}}
	actions := []webDAVSyncAction{}
	allNames := make(map[string]struct{}, len(localFiles)+len(remoteFiles)+len(state.Files))
	for name := range localFiles {
		allNames[name] = struct{}{}
	}
	for name := range remoteFiles {
		allNames[name] = struct{}{}
	}
	for name := range state.Files {
		allNames[name] = struct{}{}
	}
	names := make([]string, 0, len(allNames))
	for name := range allNames {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !isManagedDocumentFilename(name) {
			continue
		}
		local, hasLocal := localFiles[name]
		remote, hasRemote := remoteFiles[name]
		base := state.Files[name]
		localHash := ""
		if hasLocal {
			localHash = markdownHash(local.data)
		}
		remoteHash := remoteHashes[name]
		switch {
		case !hasLocal && !hasRemote:
			delete(state.Files, name)
		case base.Hash == "":
			switch {
			case hasLocal && hasRemote && localHash == remoteHash:
				state.Files[name] = webDAVFileState{Hash: localHash, ETag: remote.etag}
			case hasLocal && hasRemote:
				result.Conflicts = append(result.Conflicts, name)
			case hasLocal:
				actions = append(actions, webDAVSyncAction{kind: "upload", filename: name, data: local.data, hash: localHash})
			case hasRemote:
				actions = append(actions, webDAVSyncAction{kind: "download", filename: name, data: remoteData[name], hash: remoteHash, etag: remote.etag})
			}
		case hasLocal && hasRemote && localHash == remoteHash:
			state.Files[name] = webDAVFileState{Hash: localHash, ETag: remote.etag}
		case hasLocal && hasRemote && localHash == base.Hash:
			actions = append(actions, webDAVSyncAction{kind: "download", filename: name, data: remoteData[name], hash: remoteHash, etag: remote.etag})
		case hasLocal && hasRemote && remoteHash == base.Hash:
			actions = append(actions, webDAVSyncAction{kind: "upload", filename: name, data: local.data, hash: localHash})
		case !hasLocal && hasRemote && remoteHash == base.Hash:
			actions = append(actions, webDAVSyncAction{kind: "delete-remote", filename: name})
		case hasLocal && !hasRemote && localHash == base.Hash:
			actions = append(actions, webDAVSyncAction{kind: "delete-local", filename: name})
		case hasLocal && hasRemote:
			result.Conflicts = append(result.Conflicts, name)
		case hasLocal && !hasRemote:
			result.Conflicts = append(result.Conflicts, name)
		case !hasLocal && hasRemote:
			result.Conflicts = append(result.Conflicts, name)
		}
	}
	for _, action := range actions {
		switch action.kind {
		case "upload":
			etag, err := client.write(config.RemotePath, action.filename, action.data)
			if err != nil {
				return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, err)
			}
			state.Files[action.filename] = webDAVFileState{Hash: action.hash, ETag: etag}
			result.Uploaded++
		case "download":
			if err := r.writeLocalSyncFile(action.filename, action.data); err != nil {
				return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, err)
			}
			state.Files[action.filename] = webDAVFileState{Hash: action.hash, ETag: action.etag}
			result.Downloaded++
		case "delete-remote":
			if err := client.remove(config.RemotePath, action.filename); err != nil {
				return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, err)
			}
			delete(state.Files, action.filename)
			result.Deleted++
		case "delete-local":
			if err := r.deleteLocalSyncFile(action.filename); err != nil {
				return WebDAVSyncResult{}, r.failure("webdav_sync", apperr.WebDAVSync, err)
			}
			delete(state.Files, action.filename)
			result.Deleted++
		}
	}
	state.LastSyncedAt = time.Now().UTC().Format(time.RFC3339)
	result.LastSyncedAt = state.LastSyncedAt
	if err := r.writeWebDAVState(state); err != nil {
		return WebDAVSyncResult{}, r.failure("webdav_state_write", apperr.StorageWrite, err)
	}
	r.logger.Info("webdav_sync_completed", "uploaded", result.Uploaded, "downloaded", result.Downloaded,
		"deleted", result.Deleted, "conflicts", len(result.Conflicts))
	return result, nil
}

type localSyncFile struct {
	data []byte
}

func (r *Resume) readLocalSyncFiles() (map[string]localSyncFile, error) {
	dir := filepath.Dir(r.paths.ContentFile)
	files := map[string]localSyncFile{}
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !isManagedDocumentFilename(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		files[name] = localSyncFile{data: data}
	}
	general := filepath.Base(r.paths.ContentFile)
	if _, exists := files[general]; !exists && general == "简历内容.md" {
		if _, err := os.Stat(r.generalDeletionPath()); os.IsNotExist(err) {
			files[general] = localSyncFile{data: []byte(r.resources.DocumentMD)}
		} else if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func isManagedDocumentFilename(filename string) bool {
	return filename == "简历内容.md" || (strings.HasPrefix(filename, "简历_") && strings.HasSuffix(filename, ".md") && !strings.ContainsAny(filename, `/\\:`))
}

func markdownHash(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func (r *Resume) validateSyncDocument(filename string, data []byte) error {
	if !isManagedDocumentFilename(filename) {
		return errors.New("unmanaged Markdown filename")
	}
	content, err := ParseContent(string(data))
	if err != nil {
		return err
	}
	return r.template.Validate(content)
}

func (r *Resume) writeLocalSyncFile(filename string, data []byte) error {
	if err := r.validateSyncDocument(filename, data); err != nil {
		return err
	}
	path, err := r.documentPath(filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := writeDocumentFile(path, data); err != nil {
		return err
	}
	if filename == "简历内容.md" {
		if err := os.Remove(r.generalDeletionPath()); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (r *Resume) deleteLocalSyncFile(filename string) error {
	path, err := r.documentPath(filename)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if filename == "简历内容.md" {
		if err := os.MkdirAll(filepath.Dir(r.generalDeletionPath()), 0o700); err != nil {
			return err
		}
		return writeDocumentFile(r.generalDeletionPath(), []byte("true\n"))
	}
	return nil
}

func (r *Resume) localSyncFile(filename string) ([]byte, error) {
	path, err := r.documentPath(filename)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) && filename == "简历内容.md" {
		if _, markerErr := os.Stat(r.generalDeletionPath()); os.IsNotExist(markerErr) {
			return []byte(r.resources.DocumentMD), nil
		}
	}
	return data, err
}

func isWebDAVStatus(err error, want int) bool {
	var status webDAVStatusError
	return errors.As(err, &status) && status.status == want
}

// ResolveWebDAVConflict applies one explicit choice after checking both versions are still unchanged.
func (r *Resume) ResolveWebDAVConflict(filename, choice string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !isManagedDocumentFilename(filename) || (choice != "local" && choice != "remote") {
		return r.failure("webdav_conflict_resolve", apperr.WebDAVConfig, errors.New("invalid conflict choice"))
	}
	config, key, err := r.webDAVConfigKey()
	if err != nil {
		return r.failure("webdav_conflict_resolve", apperr.WebDAVConfig, err)
	}
	state, err := r.readWebDAVState(key)
	if err != nil {
		return r.failure("webdav_state_read", apperr.StorageRead, err)
	}
	client, err := newWebDAVClient(config.ServerURL, config.Username, config.Password)
	if err != nil {
		return r.failure("webdav_conflict_resolve", apperr.WebDAVConfig, err)
	}
	localFiles, err := r.readLocalSyncFiles()
	if err != nil {
		return r.failure("webdav_conflict_resolve", apperr.WebDAVSync, err)
	}
	localFile, hasLocal := localFiles[filename]
	remoteEntries, err := client.readDir(config.RemotePath)
	if err != nil {
		return r.failure("webdav_conflict_resolve", apperr.WebDAVSync, err)
	}
	remoteEntry, found := findWebDAVEntry(remoteEntries, filename)
	var remoteData []byte
	if found {
		remoteData, err = client.read(config.RemotePath, filename)
		if err != nil {
			return r.failure("webdav_conflict_resolve", apperr.WebDAVSync, err)
		}
	}
	base := state.Files[filename]
	localHash, remoteHash := "", ""
	if hasLocal {
		localHash = markdownHash(localFile.data)
	}
	if found {
		remoteHash = markdownHash(remoteData)
	}
	if hasLocal == found && (!hasLocal || localHash == remoteHash) {
		return r.failure("webdav_conflict_resolve", apperr.WebDAVConflict, errors.New("versions no longer conflict"))
	}
	if hasLocal && found && base.Hash != "" && (localHash == base.Hash || remoteHash == base.Hash) {
		return r.failure("webdav_conflict_resolve", apperr.WebDAVConflict, errors.New("versions no longer conflict"))
	}
	var etag string
	if choice == "local" {
		if hasLocal {
			etag, err = client.write(config.RemotePath, filename, localFile.data)
		} else if found {
			err = client.remove(config.RemotePath, filename)
		}
	} else {
		if found {
			err = r.writeLocalSyncFile(filename, remoteData)
			etag = remoteEntry.etag
		} else if hasLocal {
			err = r.deleteLocalSyncFile(filename)
		}
	}
	if err != nil {
		return r.failure("webdav_conflict_resolve", apperr.WebDAVSync, err)
	}
	chosenHash := localHash
	if choice == "remote" {
		chosenHash = remoteHash
	}
	if chosenHash == "" {
		delete(state.Files, filename)
	} else {
		state.Files[filename] = webDAVFileState{Hash: chosenHash, ETag: etag}
	}
	state.LastSyncedAt = time.Now().UTC().Format(time.RFC3339)
	if err := r.writeWebDAVState(state); err != nil {
		return r.failure("webdav_state_write", apperr.StorageWrite, err)
	}
	return nil
}

func findWebDAVEntry(entries []webDAVEntry, filename string) (webDAVEntry, bool) {
	for _, entry := range entries {
		if entry.name == filename {
			return entry, true
		}
	}
	return webDAVEntry{}, false
}

// Avoid including secret configuration fields in diagnostic formatting.
func (c webDAVConfig) String() string {
	return fmt.Sprintf("WebDAVConfig{serverURL:%q, username:%q, remotePath:%q}", c.ServerURL, c.Username, c.RemotePath)
}
