package nodetree

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gopkg.in/acd.v0/internal/constants"
	"gopkg.in/acd.v0/internal/log"
)

type (
	// Nodes is a slice of nodes
	Nodes []*Node

	// Node represents a digital asset on the Amazon Cloud Drive, including files
	// and folders, in a parent-child relationship. A node contains only metadata
	// (e.g. folder) or it contains metadata and content (e.g. file).
	Node struct {
		// Coming from Amazon
		ID                string    `json:"id,omitempty"`
		Name              string    `json:"name,omitempty"`
		Kind              string    `json:"kind,omitempty"`
		Parents           []string  `json:"Parents,omitempty"`
		Status            string    `json:"status,omitempty"`
		Labels            []string  `json:"labels,omitempty"`
		CreatedBy         string    `json:"createdBy,omitempty"`
		CreationDate      time.Time `json:"creationDate,omitempty"`
		ModifiedDate      time.Time `json:"modifiedDate,omitempty"`
		Version           uint64    `json:"version,omitempty"`
		TempLink          string    `json:"tempLink,omitempty"`
		ContentProperties struct {
			Version     uint64    `json:"version,omitempty"`
			Extension   string    `json:"extension,omitempty"`
			Size        uint64    `json:"size,omitempty"`
			MD5         string    `json:"md5,omitempty"`
			ContentType string    `json:"contentType,omitempty"`
			ContentDate time.Time `json:"contentDate,omitempty"`
		} `json:"contentProperties,omitempty"`

		// Internal
		Nodes  Nodes `json:"nodes,omitempty"`
		Root   bool  `json:"root,omitempty"`
		client client
	}

	newNode struct {
		Name       string            `json:"name,omitempty"`
		Kind       string            `json:"kind,omitempty"`
		Labels     []string          `json:"labels,omitempty"`
		Properties map[string]string `json:"properties"`
		Parents    []string          `json:"parents"`
	}

	client interface {
		GetMetadataURL(string) string
		GetContentURL(string) string
		Do(*http.Request) (*http.Response, error)
		CheckResponse(*http.Response) error
		GetNodeTree() *NodeTree
		GetTimeout() time.Duration
	}
)

// IsFile returns whether the node represents a file.
func (n *Node) IsFile() bool {
	return n.Kind == "FILE"
}

// IsFolder returns whether the node represents a folder.
func (n *Node) IsFolder() bool {
	return n.Kind == "FOLDER"
}

// Available returns true if the node is available
func (n *Node) Available() bool {
	return n.Status == "AVAILABLE"
}

// Download returns an io.ReadCloser. The caller is responsible for closing the
// body.
func (n *Node) Download() (io.ReadCloser, error) {
	url := n.client.GetContentURL(fmt.Sprintf("nodes/%s/content", n.ID))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("error creating download request: %s", err)
		return nil, constants.ErrCreatingHTTPRequest
	}

	res, err := n.client.Do(req)
	if err != nil {
		log.Errorf("error downloading the file: %s", err)
		return nil, constants.ErrDoingHTTPRequest
	}

	return res.Body, nil
}

// AddChild add a new child for the node
func (n *Node) AddChild(child *Node) {
	log.Debugf("adding %s under %s", child.Name, n.Name)
	n.Nodes = append(n.Nodes, child)
	child.client = n.client
}

// RemoveChild remove a new child for the node
func (n *Node) RemoveChild(child *Node) {
	found := false

	for i, n := range n.Nodes {
		if n == child {
			if i < len(n.Nodes)-1 {
				copy(n.Nodes[i:], n.Nodes[i+1:])
			}
			n.Nodes[len(n.Nodes)-1] = nil
			n.Nodes = n.Nodes[:len(n.Nodes)-1]
			found = true
			break
		}
	}
	log.Debugf("removing %s from %s: %t", child.Name, n.Name, found)
}

func (n *Node) update(newNode *Node) error {
	// encode the newNode to JSON and back
	v, err := json.Marshal(newNode)
	if err != nil {
		log.Errorf("error encoding the node to JSON: %s", err)
		return constants.ErrJSONEncoding
	}

	// decode it back to n
	if err := json.Unmarshal(v, n); err != nil {
		log.Errorf("error decoding the node from JSON: %s", err)
		return constants.ErrJSONDecoding
	}

	return nil
}
