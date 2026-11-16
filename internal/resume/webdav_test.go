package resume

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

type webDAVTestServer struct {
	documents   map[string][]byte
	directories map[string]bool
	getCount    int
}

type memoryWebDAVCredentialStore struct {
	values map[string]string
	setErr error
	getErr error
	delErr error
}

func newMemoryWebDAVCredentialStore() *memoryWebDAVCredentialStore {
	return &memoryWebDAVCredentialStore{values: map[string]string{}}
}

func (s *memoryWebDAVCredentialStore) Get(service, account string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	value, ok := s.values[service+"\x00"+account]
	if !ok {
		return "", errWebDAVCredentialNotFound
	}
	return value, nil
}

func (s *memoryWebDAVCredentialStore) Set(service, account, password string) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.values[service+"\x00"+account] = password
	return nil
}

func (s *memoryWebDAVCredentialStore) Delete(service, account string) error {
	if s.delErr != nil {
		return s.delErr
	}
	key := service + "\x00" + account
	if _, ok := s.values[key]; !ok {
		return errWebDAVCredentialNotFound
	}
	delete(s.values, key)
	return nil
}

func newWebDAVTestServer() (*httptest.Server, *webDAVTestServer) {
	state := &webDAVTestServer{
		documents:   map[string][]byte{},
		directories: map[string]bool{"/dav": true},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "account" || password != "app-password" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		name := r.URL.Path
		switch r.Method {
		case "MKCOL":
			if state.directories[name] {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			state.directories[name] = true
			w.WriteHeader(http.StatusCreated)
		case "PROPFIND":
			if !state.directories[name] {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(207)
			fmt.Fprint(w, `<?xml version="1.0"?><d:multistatus xmlns:d="DAV:">`)
			writeResponse := func(href, etag string, collection bool) {
				fmt.Fprintf(w, `<d:response><d:href>%s</d:href><d:propstat><d:prop><d:resourcetype>`, href)
				if collection {
					fmt.Fprint(w, `<d:collection/>`)
				}
				fmt.Fprintf(w, `</d:resourcetype><d:getetag>%s</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`, etag)
			}
			writeResponse(name+"/", "", true)
			files := make([]string, 0)
			for file := range state.documents {
				if path.Dir(file) == name {
					files = append(files, file)
				}
			}
			sort.Strings(files)
			for _, file := range files {
				etag := fmt.Sprintf(`"%x"`, markdownHash(state.documents[file]))
				writeResponse(name+"/"+url.PathEscape(path.Base(file)), etag, false)
			}
			fmt.Fprint(w, `</d:multistatus>`)
		case http.MethodGet:
			data, exists := state.documents[name]
			if !exists {
				http.NotFound(w, r)
				return
			}
			state.getCount++
			w.Write(data)
		case http.MethodPut:
			data, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "read failed", http.StatusInternalServerError)
				return
			}
			state.documents[name] = data
			w.Header().Set("ETag", fmt.Sprintf(`"%x"`, markdownHash(data)))
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			delete(state.documents, name)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	return server, state
}

func newWebDAVTestResume(t *testing.T) *Resume {
	t.Helper()
	root := t.TempDir()
	service, err := New(Resources{
		TemplateHTML: `<p data-f="name"></p>`,
		DocumentMD:   "@@name\n默认姓名\n",
		Schema:       []FieldMeta{{Key: "name", Group: "personal"}},
	}, StoragePaths{
		ContentFile: filepath.Join(root, "简历内容.md"),
		OutputDir:   filepath.Join(root, "output"),
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	service.webDAVCredentials = newMemoryWebDAVCredentialStore()
	return service
}

func saveWebDAVTestSettings(t *testing.T, service *Resume, serverURL string) {
	t.Helper()
	if _, err := service.SaveWebDAVSettings(WebDAVSettingsInput{
		ServerURL: serverURL + "/dav", Username: "account", Password: "app-password", RemotePath: "resume-editor",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWebDAVSettingsDoNotReturnPassword(t *testing.T) {
	server, _ := newWebDAVTestServer()
	defer server.Close()
	service := newWebDAVTestResume(t)
	saveWebDAVTestSettings(t, service, server.URL)
	settings, err := service.GetWebDAVSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.ServerURL != server.URL+"/dav" || settings.Username != "account" || !settings.HasPassword {
		t.Fatalf("unexpected public settings: %+v", settings)
	}
	if strings.Contains(fmt.Sprintf("%+v", settings), "app-password") {
		t.Fatal("settings exposed the stored password")
	}
	if _, err := service.SaveWebDAVSettings(WebDAVSettingsInput{
		ServerURL: settings.ServerURL, Username: settings.Username, RemotePath: settings.RemotePath,
	}); err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(service.webDAVConfigPath())
	if err != nil || strings.Contains(string(stored), "app-password") || strings.Contains(string(stored), `"password"`) {
		t.Fatalf("settings file contains a password: %s %v", stored, err)
	}
	password, err := service.webDAVCredentials.Get(webDAVCredentialService, webDAVCredentialAccount(webDAVConfig{
		ServerURL: settings.ServerURL, Username: settings.Username,
	}))
	if err != nil || password != "app-password" {
		t.Fatalf("blank password did not preserve the credential-store value: %q %v", password, err)
	}
}

func TestWebDAVSettingsFailWhenCredentialStoreIsUnavailable(t *testing.T) {
	service := newWebDAVTestResume(t)
	store := service.webDAVCredentials.(*memoryWebDAVCredentialStore)
	store.setErr = errors.New("credential store unavailable")
	if _, err := service.SaveWebDAVSettings(WebDAVSettingsInput{
		ServerURL: "https://example.com/dav", Username: "account", Password: "app-password",
	}); err == nil {
		t.Fatal("settings saved without a working credential store")
	}
	if _, err := os.Stat(service.webDAVConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("settings file was written despite credential-store failure: %v", err)
	}
}

func TestWebDAVSettingsCanClearStoredPassword(t *testing.T) {
	service := newWebDAVTestResume(t)
	if _, err := service.SaveWebDAVSettings(WebDAVSettingsInput{
		ServerURL: "https://example.com/dav", Username: "account", Password: "app-password",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveWebDAVSettings(WebDAVSettingsInput{
		ServerURL: "https://example.com/dav", Username: "account", ClearPassword: true,
	}); err != nil {
		t.Fatal(err)
	}
	settings, err := service.GetWebDAVSettings()
	if err != nil || settings.HasPassword {
		t.Fatalf("password was not cleared: %+v %v", settings, err)
	}
	stored, err := os.ReadFile(service.webDAVConfigPath())
	if err != nil || strings.Contains(string(stored), "app-password") {
		t.Fatalf("settings file contains a password after clearing: %s %v", stored, err)
	}
}

func TestWebDAVSettingsKeepCredentialsForMultipleServers(t *testing.T) {
	service := newWebDAVTestResume(t)
	if _, err := service.SaveWebDAVSettings(WebDAVSettingsInput{
		ServerURL: "https://first.example/dav", Username: "account", Password: "first-password",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveWebDAVSettings(WebDAVSettingsInput{
		ServerURL: "https://second.example/dav", Username: "account", Password: "second-password",
	}); err != nil {
		t.Fatal(err)
	}
	settings, err := service.SaveWebDAVSettings(WebDAVSettingsInput{
		ServerURL: "https://first.example/dav", Username: "account",
	})
	if err != nil || !settings.HasPassword {
		t.Fatalf("switching back did not restore the saved credential: %+v %v", settings, err)
	}
	for _, test := range []struct {
		serverURL string
		password  string
	}{
		{"https://first.example/dav", "first-password"},
		{"https://second.example/dav", "second-password"},
	} {
		password, err := service.webDAVCredentials.Get(webDAVCredentialService, webDAVCredentialAccount(webDAVConfig{
			ServerURL: test.serverURL, Username: "account",
		}))
		if err != nil || password != test.password {
			t.Fatalf("credential for %s was not retained: %q %v", test.serverURL, password, err)
		}
	}
}

func TestWebDAVSyncTransfersChangesAndResolvesConflicts(t *testing.T) {
	server, remote := newWebDAVTestServer()
	defer server.Close()
	service := newWebDAVTestResume(t)
	saveWebDAVTestSettings(t, service, server.URL)

	first, err := service.SyncWebDAV()
	if err != nil || first.Uploaded != 1 || len(first.Conflicts) != 0 {
		t.Fatalf("initial sync: %+v %v", first, err)
	}
	if string(remote.documents["/dav/resume-editor/简历内容.md"]) != "@@name\n默认姓名\n" {
		t.Fatalf("default resume was not uploaded: %q", remote.documents["/dav/resume-editor/简历内容.md"])
	}

	localPath := filepath.Join(filepath.Dir(service.paths.ContentFile), "简历内容.md")
	if err := writeDocumentFile(localPath, []byte("@@name\n本机修改\n")); err != nil {
		t.Fatal(err)
	}
	second, err := service.SyncWebDAV()
	if err != nil || second.Uploaded != 1 || string(remote.documents["/dav/resume-editor/简历内容.md"]) != "@@name\n本机修改\n" {
		t.Fatalf("local update was not uploaded: %+v %v", second, err)
	}

	remote.documents["/dav/resume-editor/简历内容.md"] = []byte("@@name\n云端修改\n")
	third, err := service.SyncWebDAV()
	if err != nil || third.Downloaded != 1 {
		t.Fatalf("remote update was not downloaded: %+v %v", third, err)
	}
	local, err := os.ReadFile(localPath)
	if err != nil || string(local) != "@@name\n云端修改\n" {
		t.Fatalf("downloaded file differs: %q %v", local, err)
	}

	if err := writeDocumentFile(localPath, []byte("@@name\n本机分歧\n")); err != nil {
		t.Fatal(err)
	}
	remote.documents["/dav/resume-editor/简历内容.md"] = []byte("@@name\n云端分歧\n")
	conflicted, err := service.SyncWebDAV()
	if err != nil || len(conflicted.Conflicts) != 1 || conflicted.Conflicts[0] != "简历内容.md" {
		t.Fatalf("concurrent updates were not reported: %+v %v", conflicted, err)
	}
	if err := service.ResolveWebDAVConflict("简历内容.md", "remote"); err != nil {
		t.Fatal(err)
	}
	resolved, err := os.ReadFile(localPath)
	if err != nil || string(resolved) != "@@name\n云端分歧\n" {
		t.Fatalf("remote choice was not applied: %q %v", resolved, err)
	}
	last, err := service.SyncWebDAV()
	if err != nil || len(last.Conflicts) != 0 {
		t.Fatalf("conflict remained after resolution: %+v %v", last, err)
	}
}

func TestWebDAVSyncPropagatesFileDeletions(t *testing.T) {
	server, remote := newWebDAVTestServer()
	defer server.Close()
	service := newWebDAVTestResume(t)
	saveWebDAVTestSettings(t, service, server.URL)
	if _, err := service.SyncWebDAV(); err != nil {
		t.Fatal(err)
	}
	localPath := filepath.Join(filepath.Dir(service.paths.ContentFile), "简历内容.md")
	if err := writeDocumentFile(localPath, []byte("@@name\n默认姓名\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(localPath); err != nil {
		t.Fatal(err)
	}
	if err := writeDocumentFile(service.generalDeletionPath(), []byte("true\n")); err != nil {
		t.Fatal(err)
	}
	result, err := service.SyncWebDAV()
	if err != nil || result.Deleted != 1 {
		t.Fatalf("local deletion was not propagated: %+v %v", result, err)
	}
	if len(remote.documents) != 0 {
		t.Fatalf("remote file remained after deletion: %+v", remote.documents)
	}
}

func TestWebDAVDeleteModifyConflictsCanBeResolved(t *testing.T) {
	const filename = "简历_演示_工程师.md"
	const remotePath = "/dav/resume-editor/" + filename
	base := []byte("@@name\n基线\n")
	for _, test := range []struct {
		name       string
		local      []byte
		remote     []byte
		choice     string
		wantLocal  bool
		wantRemote bool
	}{
		{name: "local delete kept", remote: []byte("@@name\n云端修改\n"), choice: "local"},
		{name: "remote edit kept", remote: []byte("@@name\n云端修改\n"), choice: "remote", wantLocal: true, wantRemote: true},
		{name: "local edit kept", local: []byte("@@name\n本机修改\n"), choice: "local", wantLocal: true, wantRemote: true},
		{name: "remote delete kept", local: []byte("@@name\n本机修改\n"), choice: "remote"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, remote := newWebDAVTestServer()
			defer server.Close()
			service := newWebDAVTestResume(t)
			saveWebDAVTestSettings(t, service, server.URL)
			localPath := filepath.Join(filepath.Dir(service.paths.ContentFile), filename)
			if err := writeDocumentFile(localPath, base); err != nil {
				t.Fatal(err)
			}
			if _, err := service.SyncWebDAV(); err != nil {
				t.Fatal(err)
			}
			if len(test.local) == 0 {
				if err := os.Remove(localPath); err != nil {
					t.Fatal(err)
				}
			} else if err := writeDocumentFile(localPath, test.local); err != nil {
				t.Fatal(err)
			}
			if len(test.remote) == 0 {
				delete(remote.documents, remotePath)
			} else {
				remote.documents[remotePath] = test.remote
			}
			result, err := service.SyncWebDAV()
			if err != nil || len(result.Conflicts) != 1 || result.Conflicts[0] != filename {
				t.Fatalf("delete-modify conflict was not reported: %+v %v", result, err)
			}
			if err := service.ResolveWebDAVConflict(filename, test.choice); err != nil {
				t.Fatal(err)
			}
			_, localErr := os.Stat(localPath)
			if localErr != nil && !os.IsNotExist(localErr) {
				t.Fatal(localErr)
			}
			_, remoteExists := remote.documents[remotePath]
			if (localErr == nil) != test.wantLocal || remoteExists != test.wantRemote {
				t.Fatalf("resolution local=%v remote=%v, want local=%v remote=%v", localErr == nil, remoteExists, test.wantLocal, test.wantRemote)
			}
			result, err = service.SyncWebDAV()
			if err != nil || len(result.Conflicts) != 0 {
				t.Fatalf("conflict remained after resolution: %+v %v", result, err)
			}
		})
	}
}

func TestWebDAVSyncDoesNotTreatRemovedDirectoryAsMassDeletion(t *testing.T) {
	server, remote := newWebDAVTestServer()
	defer server.Close()
	service := newWebDAVTestResume(t)
	saveWebDAVTestSettings(t, service, server.URL)
	if _, err := service.SyncWebDAV(); err != nil {
		t.Fatal(err)
	}
	remote.directories["/dav/resume-editor"] = false
	if _, err := service.SyncWebDAV(); err == nil {
		t.Fatal("sync accepted removal of a previously used remote directory")
	}
	if len(remote.documents) != 1 {
		t.Fatal("removed remote directory caused a file deletion")
	}
}

func TestWebDAVDirectoryPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Link", `</dav/resume-editor?page=2>; rel="next"`)
		w.WriteHeader(207)
		_, _ = io.WriteString(w, `<?xml version="1.0"?><d:multistatus xmlns:d="DAV:"><d:response><d:href>/dav/resume-editor/</d:href><d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response><d:response><d:href>/dav/resume-editor/%E7%AE%80%E5%8E%86%E5%86%85%E5%AE%B9.md</d:href><d:propstat><d:prop><d:resourcetype/><d:getetag>one</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`)
	}))
	defer server.Close()
	client, err := newWebDAVClient(server.URL+"/dav", "account", "app-password")
	if err != nil {
		t.Fatal(err)
	}
	client.http = server.Client()
	client.http.Timeout = 5 * time.Second
	client.http.CheckRedirect = func(request *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if strings.Contains(request.URL.RawQuery, "page=2") {
			response := &http.Response{StatusCode: 207, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`<?xml version="1.0"?><d:multistatus xmlns:d="DAV:"><d:response><d:href>/dav/resume-editor/简历_第二份.md</d:href><d:propstat><d:prop><d:resourcetype/><d:getetag>two</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`))}
			return response, nil
		}
		return http.DefaultTransport.RoundTrip(request)
	})
	entries, err := client.readDir("resume-editor")
	if err != nil || len(entries) != 2 {
		t.Fatalf("pagination omitted entries: %+v %v", entries, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
