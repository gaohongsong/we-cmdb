package graph

import "github.com/WeBankPartners/we-cmdb/cmdb-server/models"

type MapString map[string]string
type MapData map[string]interface{}

type RenderOption struct {
	SuportVersion string            `json:"suport_version"`
	ImageMap      map[string]string `json:"image_map"`
}

type MetaData struct {
	GraphType     string    `json:"graph_type"`
	GraphDir      string    `json:"graph_dir"`
	ConfirmTime   string    `json:"confirm_time"`
	FontSize      float64   `json:"fontSize"`
	FontStep      float64   `json:"font_step"`
	SuportVersion string    `json:"suport_version"`
	ImagesMap     MapString `json:"imagesMap"`
	RenderedItems []string  `json:"renderedItems"`
}

type Line struct {
	Setting  *models.GraphElementNode
	DataList []MapData
	MetaData MetaData
}

type RenderResult struct {
	DotString     string
	Lines         []Line
	RenderedItems []string
	Error         error
}

// Element 暂时没用到，后续考虑将map部分结构化
type Element struct {
	Code        string `json:"code"`
	Guid        string `json:"guid"`
	KeyName     string `json:"key_name"`
	State       string `json:"state"`
	ConfirmTime string `json:"confirm_time,omitempty"`
	UpdateTime  string `json:"update_time,omitempty"`
}
