package ns

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// LogIn logs in to NexentaStor API and get auth token
func (p *Provider) LogIn() error {
	l := p.Log.WithField("func", "LogIn()")

	data := nefAuthLoginRequest{
		Username: p.Username,
		Password: p.Password,
	}

	_, bodyBytes, err := p.RestClient.Send(http.MethodPost, "auth/login", data)
	if err != nil {
		// try to parse error from rest response
		nefError := p.parseNefError(bodyBytes, "Login request")
		if nefError != nil {
			if IsAuthNefError(nefError) {
				l.Errorf(
					"login to NexentaStor %s failed (username: '%s'), "+
						"please make sure to use correct address and password",
					p.Address,
					p.Username)
			}
			return nefError
		}

		return fmt.Errorf("Login request: failed, response: %s; error: %s", bodyBytes, err)
	}

	response := nefAuthLoginResponse{}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return fmt.Errorf("Login request: cannot unmarshal JSON from: '%s' to '%+v': %s", bodyBytes, response, err)
	} else if response.Token == "" {
		return fmt.Errorf("Login request: token not found in response: '%s'", bodyBytes)
	}

	p.RestClient.SetAuthToken(response.Token)
	l.Debugf("login token has been updated")
	return nil
}

// GetLicense returns NexentaStor license
func (p *Provider) GetLicense() (license License, err error) {
	err = p.sendRequestWithStruct(http.MethodGet, "/settings/license", nil, &license)
	return license, err
}

// GetPools returns NexentaStor pools
func (p *Provider) GetPools() ([]Pool, error) {
	uri := p.RestClient.BuildURI("/storage/pools", map[string]string{
		"fields": "poolName,health,status",
	})

	response := nefStoragePoolsResponse{}
	err := p.sendRequestWithStruct(http.MethodGet, uri, nil, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

// GetFilesystemAvailableCapacity returns NexentaStor filesystem available size by its path
func (p *Provider) GetFilesystemAvailableCapacity(path string) (int64, error) {
	uri := p.RestClient.BuildURI("/storage/filesystems", map[string]string{
		"path":   path,
		"fields": "bytesAvailable",
	})

	response := nefStorageFilesystemsResponse{}
	err := p.sendRequestWithStruct(http.MethodGet, uri, nil, &response)
	if err != nil {
		return 0, err
	}

	var availableSize int64
	if len(response.Data) > 0 {
		availableSize = int64(response.Data[0].BytesAvailable)
	}

	return availableSize, nil
}

// GetFilesystem returns NexentaStor filesystem by its path
func (p *Provider) GetFilesystem(path string) (filesystem Filesystem, err error) {
	if path == "" {
		return filesystem, fmt.Errorf("Filesystem path is empty")
	}

	uri := p.RestClient.BuildURI("/storage/filesystems", map[string]string{
		"path":   path,
		"fields": "path,mountPoint,bytesAvailable,bytesUsed,sharedOverNfs,sharedOverSmb",
	})

	response := nefStorageFilesystemsResponse{}
	err = p.sendRequestWithStruct(http.MethodGet, uri, nil, &response)
	if err != nil {
		return filesystem, err
	}

	if len(response.Data) == 0 {
		return filesystem, &NefError{Code: "ENOENT", Err: fmt.Errorf("Filesystem '%s' not found", path)}
	}

	return response.Data[0], nil
}

// GetFilesystems returns all NexentaStor filesystems by parent filesystem
func (p *Provider) GetFilesystems(parent string) ([]Filesystem, error) {
	uri := p.RestClient.BuildURI("/storage/filesystems", map[string]string{
		"parent": parent,
		"fields": "path,mountPoint,bytesAvailable,bytesUsed,sharedOverNfs,sharedOverSmb",
	})

	response := nefStorageFilesystemsResponse{}
	err := p.sendRequestWithStruct(http.MethodGet, uri, nil, &response)
	if err != nil {
		return nil, err
	}

	filesystems := []Filesystem{}
	for _, fs := range response.Data {
		if fs.Path != parent { // exclude parent filesystem from the list
			filesystems = append(filesystems, fs)
		}
	}

	return filesystems, nil
}

// CreateFilesystemParams - params to create filesystem
type CreateFilesystemParams struct {
	// filesystem path w/o leading slash
	Path string `json:"path"`
	// filesystem referenced quota size in bytes
	ReferencedQuotaSize int64 `json:"referencedQuotaSize,omitempty"`
}

// CreateFilesystem creates filesystem by path
func (p *Provider) CreateFilesystem(params CreateFilesystemParams) error {
	if params.Path == "" {
		return fmt.Errorf("Parameter 'CreateFilesystemParams.Path' is required")
	}

	//TODO consider to add option https://jira.nexenta.com/browse/NEX-17476?focusedCommentId=154590

	return p.sendRequest(http.MethodPost, "/storage/filesystems", params)
}

// DestroyFilesystem destroys filesystem by path
func (p *Provider) DestroyFilesystem(path string, destroySnapshots bool) error {
	if path == "" {
		return fmt.Errorf("Filesystem path is required")
	}

	destroySnapshotsString := "false"
	if destroySnapshots {
		destroySnapshotsString = "true"
	}

	uri := p.RestClient.BuildURI(
		fmt.Sprintf("/storage/filesystems/%s", url.PathEscape(path)),
		map[string]string{
			"force":     "true",
			"snapshots": destroySnapshotsString,
		},
	)

	return p.sendRequest(http.MethodDelete, uri, nil)
}

// DestroyFilesystemWithClones destroys filesystem that has or may have clones
// Method promotes filesystem's most recent snapshots and then delete the filesystem
func (p *Provider) DestroyFilesystemWithClones(path string, destroySnapshots bool) (err error) {
	l := p.Log.WithField("func", "DestroyFilesystemWithClones()")

	maxAttemptCount := 3

	for i := 0; i < maxAttemptCount; i++ {
		err = p.DestroyFilesystem(path, destroySnapshots)
		if err == nil || IsNotExistNefError(err) {
			return err
		} else if !IsAlreadyExistNefError(err) {
			continue
		}

		// if here then filesystem has dependent clones (EEXIST error on deletion),
		// need promote the most recent clone to make the filesystem independent

		snapshots, err := p.GetSnapshots(path, true)
		if err != nil {
			l.Warnf("filesystem '%s' has clones, but snapshots list was failed: %s (trying again...)", path, err)
			continue
		}

		var maxCreationTxg int
		var mostRecentClone string
		for _, s := range snapshots {
			// to get "clones" and "creationTxg" fields that are not presented int the list response
			snapshot, err := p.GetSnapshot(s.Path)
			if err != nil {
				l.Warnf(
					"filesystem '%s' has clones, but getting snapshot '%s' was failed: %s (trying again...)",
					path,
					s.Path,
					err,
				)
				continue
			}
			creationTxg, err := strconv.Atoi(snapshot.CreationTxg)
			if len(snapshot.Clones) > 0 && creationTxg > maxCreationTxg {
				mostRecentClone = snapshot.Clones[0]
			}
		}

		if mostRecentClone == "" {
			l.Warnf("filesystem '%s' is supposted to has clones, but it doesn't, trying again...", path)
			continue
		}

		l.Infof("filesystem '%s' has clone '%s', promote it to free up the filesystem...", path, mostRecentClone)
		err = p.PromoteFilesystem(mostRecentClone)
		if err != nil {
			l.Warnf("failed to promote filesystem clone '%s': %v (trying again...)", mostRecentClone, err)
		}
	}

	return fmt.Errorf("Failed to delete filesystem '%s' x%d times, last error: %v", path, maxAttemptCount, err)
}

// PromoteFilesystem promotes a cloned filesystem to be no longer dependent on its original snapshot
func (p *Provider) PromoteFilesystem(path string) error {
	if path == "" {
		return fmt.Errorf("Filesystem path is required")
	}

	uri := fmt.Sprintf("/storage/filesystems/%s/promote", url.PathEscape(path))

	return p.sendRequest(http.MethodPost, uri, nil)
}

// CreateNfsShareParams - params to create NFS share
type CreateNfsShareParams struct {
	// filesystem path w/o leading slash
	Filesystem string `json:"filesystem"`
}

// CreateNfsShare creates NFS share on specified filesystem
// CLI test:
//	 showmount -e HOST
// 	 mkdir -p /mnt/test && sudo mount -v -t nfs HOST:/pool/fs /mnt/test
// 	 findmnt /mnt/test
func (p *Provider) CreateNfsShare(params CreateNfsShareParams) error {
	if params.Filesystem == "" {
		return fmt.Errorf("CreateNfsShareParams.Filesystem is required")
	}

	data := nefNasNfsRequest{
		Filesystem: params.Filesystem,
		Anon:       "root",
		SecurityContexts: []nefNasNfsRequestSecurityContext{
			nefNasNfsRequestSecurityContext{
				SecurityModes: []string{"sys"},
			},
		},
	}

	return p.sendRequest(http.MethodPost, "nas/nfs", data)
}

// DeleteNfsShare destroys NFS chare by filesystem path
func (p *Provider) DeleteNfsShare(path string) error {
	if path == "" {
		return fmt.Errorf("Filesystem path is empty")
	}

	uri := fmt.Sprintf("/nas/nfs/%s", url.PathEscape(path))

	return p.sendRequest(http.MethodDelete, uri, nil)
}

// CreateSmbShareParams - params to create SMB share
type CreateSmbShareParams struct {
	// filesystem path w/o leading slash
	Filesystem string `json:"filesystem"`
	// share name, used in mount command
	ShareName string `json:"shareName,omitempty"`
}

// CreateSmbShare creates SMB share (cifs) on specified filesystem
// Leave shareName empty to generate default value
// CLI test:
// 	 mkdir -p /mnt/test && sudo mount -v -t cifs -o username=admin,password=Nexenta@1 //HOST//pool_fs /mnt/test
// 	 findmnt /mnt/test
func (p *Provider) CreateSmbShare(params CreateSmbShareParams) error {
	if params.Filesystem == "" {
		return fmt.Errorf("CreateSmbShareParams.Filesystem is required")
	}

	return p.sendRequest(http.MethodPost, "nas/smb", params)
}

// GetSmbShareName returns share name for filesystem that shared over SMB
func (p *Provider) GetSmbShareName(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("Filesystem path is required")
	}

	uri := p.RestClient.BuildURI(
		fmt.Sprintf("/nas/smb/%s", url.PathEscape(path)),
		map[string]string{"fields": "shareName,shareState"}, //TODO check shareState value?
	)

	response := nefNasSmbResponse{}
	err := p.sendRequestWithStruct(http.MethodGet, uri, nil, &response)
	if err != nil {
		return "", err
	}

	return response.ShareName, nil
}

// DeleteSmbShare destroys SMB share by filesystem path
func (p *Provider) DeleteSmbShare(path string) error {
	if path == "" {
		return fmt.Errorf("Filesystem path is empty")
	}

	uri := fmt.Sprintf("/nas/smb/%s", url.PathEscape(path))

	return p.sendRequest(http.MethodDelete, uri, nil)
}

// SetFilesystemACL sets filesystem ACL, so NFS share can allow user to write w/o checking UNIX user uid
func (p *Provider) SetFilesystemACL(path string, aclRuleSet ACLRuleSet) error {
	if path == "" {
		return fmt.Errorf("Filesystem path is required")
	}

	uri := fmt.Sprintf("/storage/filesystems/%s/acl", url.PathEscape(path))

	permissions := []string{}
	if aclRuleSet == ACLReadOnly {
		permissions = append(permissions, "read_set")
	} else {
		permissions = append(permissions, "full_set")
	}

	data := &nefStorageFilesystemsACLRequest{
		Type:      "allow",
		Principal: "everyone@",
		Flags: []string{
			"file_inherit",
			"dir_inherit",
		},
		Permissions: permissions,
	}

	return p.sendRequest(http.MethodPost, uri, data)
}

// CreateSnapshotParams - params to create snapshot
type CreateSnapshotParams struct {
	// snapshot path w/o leading slash
	Path string `json:"path"`
}

// CreateSnapshot creates snapshot by filesystem path
func (p *Provider) CreateSnapshot(params CreateSnapshotParams) error {
	if params.Path == "" {
		return fmt.Errorf("Parameter 'CreateSnapshotParams.Path' is required")
	}

	return p.sendRequest(http.MethodPost, "/storage/snapshots", params)
}

// GetSnapshot returns snapshot by its path
// path - full path to snapshot w/o leading slash (e.g. "p/d/fs@s")
func (p *Provider) GetSnapshot(path string) (snapshot Snapshot, err error) {
	if path == "" {
		return snapshot, fmt.Errorf("Snapshot path is empty")
	}

	uri := p.RestClient.BuildURI(fmt.Sprintf("/storage/snapshots/%s", url.PathEscape(path)), map[string]string{
		"fields": "path,name,parent,creationTime,clones,creationTxg",
		//TODO return "bytesReferenced" and check on volume creation
	})

	err = p.sendRequestWithStruct(http.MethodGet, uri, nil, &snapshot)

	return snapshot, err
}

// GetSnapshots returns snapshots by volume path
func (p *Provider) GetSnapshots(volumePath string, recursive bool) ([]Snapshot, error) {
	if volumePath == "" {
		return []Snapshot{}, fmt.Errorf("Snapshots volume path is empty")
	}

	recursiveString := "false"
	if recursive {
		recursiveString = "true"
	}

	uri := p.RestClient.BuildURI("/storage/snapshots", map[string]string{
		"parent":    volumePath,
		"fields":    "path,name,parent,creationTime",
		"recursive": recursiveString,
	})

	response := nefStorageSnapshotsResponse{}
	err := p.sendRequestWithStruct(http.MethodGet, uri, nil, &response)
	if err != nil {
		return []Snapshot{}, err
	}

	return response.Data, nil
}

// DestroySnapshot destroys snapshot by path
func (p *Provider) DestroySnapshot(path string) error {
	if path == "" {
		return fmt.Errorf("Snapshot path is required")
	}

	uri := fmt.Sprintf("/storage/snapshots/%s", url.PathEscape(path))

	return p.sendRequest(http.MethodDelete, uri, nil)
}

// CloneSnapshotParams - params to clone snapshot to filesystem
type CloneSnapshotParams struct {
	// filesystem path w/o leading slash
	TargetPath string `json:"targetPath"`
}

// CloneSnapshot clones snapshot to FS
func (p *Provider) CloneSnapshot(path string, params CloneSnapshotParams) error {
	if path == "" {
		return fmt.Errorf("Snapshot path is required")
	}

	if params.TargetPath == "" {
		return fmt.Errorf("Parameter 'CloneSnapshotParams.TargetPath' is required")
	}

	uri := fmt.Sprintf("/storage/snapshots/%s/clone", url.PathEscape(path))

	return p.sendRequest(http.MethodPost, uri, params)
}

// GetRSFClusters returns RSF clusters from NS
func (p *Provider) GetRSFClusters() ([]RSFCluster, error) {
	uri := p.RestClient.BuildURI("/rsf/clusters", map[string]string{
		"fields": "clusterName,nodes",
	})

	response := nefRsfClustersResponse{}
	err := p.sendRequestWithStruct(http.MethodGet, uri, nil, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

// IsJobDone checks if job is done by jobId
func (p *Provider) IsJobDone(jobID string) (bool, error) {
	uri := fmt.Sprintf("/jobStatus/%s", jobID)

	statusCode, bodyBytes, err := p.RestClient.Send(http.MethodGet, uri, nil)
	if err != nil { // request failed
		return false, err
	} else if statusCode == http.StatusOK || statusCode == http.StatusCreated { // job is completed
		return true, nil
	} else if statusCode == http.StatusAccepted { // job is in progress (202)
		return false, nil
	}

	// job is failed
	nefError := p.parseNefError(bodyBytes, "Job was finished with error")
	if nefError != nil {
		err = nefError
	} else {
		err = fmt.Errorf(
			"Job request returned %d code, but response body doesn't contain explanation: %s",
			statusCode,
			bodyBytes,
		)
	}

	return false, err
}
