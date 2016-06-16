package nvdapi

import (
	"fmt"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"os/exec"
	"path/filepath"
)


type Client struct {
	Protocol          string
	Endpoint          string
	Path              string
	DefaultVolSize    int64 //bytes
	Config            *Config
	Port 			  int64
	MountPoint		  string
	Filesystem  	  string
}

type Config struct {
	IOProtocol	string // NFS, iSCSI, NBD, S3
	IP			string // server:/export, IQN, devname, 
	Port        int64
	Pool        string
	MountPoint	string
	Filesystem  string
}

func ReadParseConfig(fname string) (Config, error) {
	content, err := ioutil.ReadFile(fname)
	var conf Config
	if err != nil {
		err = fmt.Errorf("Error processing config file: ", err)
		return conf, err
	}
	err = json.Unmarshal(content, &conf)
	if err != nil {
		err = fmt.Errorf("Error parsing config file: ", err)
	}
	return conf, err
}

func ClientAlloc(configFile string) (c *Client, err error) {
	conf, err := ReadParseConfig(configFile)
	if err != nil {
		log.Fatal("Error initializing client from Config file: ", configFile, "(", err, ")")
	}

	NexentaClient := &Client{
		Protocol: conf.IOProtocol,
		Endpoint: fmt.Sprintf("http://%s:%d/", conf.IP, conf.Port),
		Path: filepath.Join(conf.Pool, conf.Filesystem),
		Config:	&conf,
		MountPoint: conf.MountPoint,
	}

	return NexentaClient, nil
}

func (c *Client) Request(method, endpoint string, data map[string]interface{}) (body []byte, err error) {
	log.Debug("Issue request to Nexenta, endpoint: ", endpoint, " data: ", data, " method: ", method)
	if c.Endpoint == "" {
		log.Error("Endpoint is not set, unable to issue requests")
		err = errors.New("Unable to issue json-rpc requests without specifying Endpoint")
		return nil, err
	}
	datajson, err := json.Marshal(data)
	if (err != nil) {
		log.Error(err)
	}

	tr := &http.Transport{}
	client := &http.Client{Transport: tr}
	url := c.Endpoint + endpoint
	req, err := http.NewRequest(method, url, nil)
	if len(data) != 0 {
		req, err = http.NewRequest(method, url, strings.NewReader(string(datajson)))
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error while handling request", err)
		return nil, err
	}
	c.checkError(resp)
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if (err != nil) {
		log.Error(err)
	}
	if (resp.StatusCode == 202) {
		body, err = c.resend202(body)
	}
	return body, err
}

func (c *Client) resend202(body []byte) ([]byte, error) {
	time.Sleep(1000 * time.Millisecond)
	r := make(map[string][]map[string]string)
	err := json.Unmarshal(body, &r)
	if (err != nil) {
		err = fmt.Errorf("Error while trying to unmarshal json %s", err)
		return body, err
	}

	url := c.Endpoint + r["links"][0]["href"]
	resp, err := http.Get(url)
	if err != nil {
		err = fmt.Errorf("Error while handling request %s", err)
		return body, err
	}
	defer resp.Body.Close()
	c.checkError(resp)

	if resp.StatusCode == 202 {
		body, err = c.resend202(body)
	}
	body, err = ioutil.ReadAll(resp.Body)
	return body, err
}

func (c *Client) checkError(resp *http.Response) (err error) {
	if resp.StatusCode > 399 {
		body, err := ioutil.ReadAll(resp.Body)
		err = fmt.Errorf("Got error in response from Nexenta, status_code: %s, body: %s", resp.StatusCode, string(body))
		return err
	}
	return err
}

func (c *Client) CreateVolume(name string) (err error) {
	log.Debug("Creating volume %s", name)
	data := map[string]interface{} {
		"path": filepath.Join(c.Path, name),
	}
	c.Request("POST", "storage/filesystems", data)

    data = make(map[string]interface{})
    rw := map[string]interface{} {"allow": true, "etype": "fqnip", "entity": "*",}
    rwlist := []map[string]interface{} {rw}
    readWriteList := map[string]interface{} {"readWriteList": rwlist}
    data["securityContexts"] = []interface{} {readWriteList}
    data["filesystem"] = filepath.Join(c.Path, name)
	c.Request("POST", "nas/nfs", data)

    data = make(map[string]interface{})
	perms := []string {"list_directory", "read_data", "add_file", "write_data", "add_subdirectory",
		"append_data", "read_xattr", "write_xattr", "execute", "delete_child", "read_attributes",
		"write_attributes", "delete", "read_acl", "write_acl", "write_owner", "synchronize"}
	flags := []string {"file_inherit", "dir_inherit"}
	data["type"] = "allow"
	data["principal"] = "everyone@"
	data["permissions"] = perms
	data["flags"] = flags
	path := filepath.Join(c.Path, name)
	url := filepath.Join("/storage/filesystems", url.QueryEscape(path), "acl")
	_, err = c.Request("POST", url, data)
	return err
}

func (c *Client) DeleteVolume(name string) (err error) {
	log.Debug("Deleting Volume ", name)
	path := filepath.Join(c.Path, name)
	body, err := c.Request("DELETE",  filepath.Join("storage/filesystems/", url.QueryEscape(path), nil))
	if strings.Contains(string(body), "ENOENT") {
		log.Info("Could not delete volume ", name, ", probably not found.")
		log.Debug("Error trying to delete volume ", name, " :", string(body))
	}
	return err
}

func (c *Client) MountVolume(name string) (err error) {
	log.Debug("MountVolume ", name)
	args := []string{"-t", "nfs", fmt.Sprintf( + "%s:/volumes/%s", c.Config.IP, filepath.Join(c.Path, name), filepath.Join(c.MountPoint, name)}
	if out, err := exec.Command("mkdir", filepath.Join(c.MountPoint, name)).CombinedOutput(); err != nil {
		log.Info("Error running mkdir command: ", err, "{", string(out), "}")
	}
	if out, err := exec.Command("mount", args...).CombinedOutput(); err != nil {
		log.Info("Error running mount command: ", err, "{", string(out), "}")
	}
	return err
}

func (c *Client) UnmountVolume(name string) (err error) {
	log.Debug("Unmounting Volume ", name)
	path := fmt.Sprintf("%s:/volumes/%s", c.Config.IP, filepath.Join(c.Path, name))
	if out, err := exec.Command("umount", path).CombinedOutput(); err != nil {
		err = fmt.Errorf("Error running umount command: ", err, "{", string(out), "}")
		return err
	}
	log.Debug("Successfully unmounted volume: ", name)
	return err
}

func (c *Client) GetVolume(name string) (vname string, err error) {
	log.Debug("GetVolume ", name)
	url := fmt.Sprintf("/storage/filesystems?path=%s", filepath.Join(c.Path, name))
	body, err := c.Request("GET", url, nil)
	r := make(map[string][]map[string]interface{})
	jsonerr := json.Unmarshal(body, &r)
	if (jsonerr != nil) {
		log.Error(jsonerr)
	}
	if len(r["data"]) < 1 {
		err = fmt.Errorf("Failed to find any volumes with name: %s.", name)
		return vname, err
	} else {
		if v,ok := r["data"][0]["path"].(string); ok {
			vname = strings.Trim(v, c.Path + "/")
			} else {
				return "", fmt.Errorf("Path is not of type string")
		}
	}
	return vname, err
}

func (c *Client) ListVolumes() (vlist []string, err error) {
	log.Debug("ListVolumes ")
	url := fmt.Sprintf("/storage/filesystems?parent=%s", c.Path)
	resp, err := c.Request("GET", url, nil)
	r := make(map[string][]map[string]interface{})
	jsonerr := json.Unmarshal(resp, &r)
	if (jsonerr != nil) {
		log.Error(jsonerr)
	}
	if len(r["data"]) < 1 {
		err = fmt.Errorf("Failed to find any volumes in filesystem: %s.", c.Path)
		return vlist, err
	} else {
		for _, vol := range r["data"] {
			if v,ok := vol["path"].(string); ok {
				vname := strings.Trim(v, c.Path + "/")
				vlist = append(vlist, strings.Trim(vname, c.Path + "/"))
				} else {
					return []string {""}, fmt.Errorf("Path is not of type string")
			}
		}
	}
	return vlist, err
}
