package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// GetLabels retrieves labels for a page.
func (c *Client) GetLabels(pageID string) ([]Label, error) {
	if c.IsCloud() {
		data, err := c.getV2("pages/"+pageID+"/labels", nil)
		if err != nil {
			return nil, err
		}
		var page CursorPage[Label]
		if err := json.Unmarshal(data, &page); err != nil {
			return nil, fmt.Errorf("parsing labels: %w", err)
		}
		return page.Results, nil
	}

	// V1
	q := url.Values{"expand": {"metadata.labels"}}
	data, err := c.getV1("content/"+pageID, q)
	if err != nil {
		return nil, err
	}
	var v1 V1Content
	if err := json.Unmarshal(data, &v1); err != nil {
		return nil, fmt.Errorf("parsing v1 labels: %w", err)
	}
	if v1.Metadata != nil && v1.Metadata.Labels != nil {
		return v1.Metadata.Labels.Results, nil
	}
	return nil, nil
}

// AddLabels adds labels to a page.
func (c *Client) AddLabels(pageID string, labels []string) error {
	if c.IsCloud() {
		entries := make(AddLabelsRequest, len(labels))
		for i, l := range labels {
			entries[i] = AddLabelEntry{Prefix: "global", Name: l}
		}
		_, err := c.postV2("pages/"+pageID+"/labels", entries)
		return err
	}

	// V1
	entries := make([]map[string]string, len(labels))
	for i, l := range labels {
		entries[i] = map[string]string{"prefix": "global", "name": l}
	}
	_, err := c.postV1("content/"+pageID+"/label", entries)
	return err
}

// RemoveLabel removes a label from a page.
func (c *Client) RemoveLabel(pageID string, labelName string) error {
	if c.IsCloud() {
		// V2 delete requires label ID — need to look it up first.
		labels, err := c.GetLabels(pageID)
		if err != nil {
			return err
		}
		for _, l := range labels {
			if l.Name == labelName {
				_, err := c.deleteV2("pages/" + pageID + "/labels/" + l.ID)
				return err
			}
		}
		return fmt.Errorf("label %q not found on page %s", labelName, pageID)
	}

	// V1: delete by name.
	fullURL := c.v1URL("content/" + pageID + "/label/" + labelName)
	_, err := c.request("DELETE", fullURL, nil, nil)
	return err
}
