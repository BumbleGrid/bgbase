package stylemap

type StyleMap struct {
	Global  *StyleRules                      `json:"global,omitempty"`
	ByFloor map[string]StyleRules            `json:"byFloor,omitempty"`
	ByTag   map[string]map[string]StyleRules `json:"byTag,omitempty"`
	ByID    map[string]StyleRules            `json:"byId,omitempty"`
}

type StyleRules struct {
	Node *NodeStyleRules `json:"node,omitempty"`
	Edge *EdgeStyleRules `json:"edge,omitempty"`
}

type NodeStyleRules struct {
	Color       string          `json:"color,omitempty"`
	TextColor   string          `json:"textColor,omitempty"`
	BorderColor string          `json:"borderColor,omitempty"`
	BorderWidth float64         `json:"borderWidth,omitempty"`
	BorderStyle string          `json:"borderStyle,omitempty"`
	Shape       string          `json:"shape,omitempty"`
	Icon        string          `json:"icon,omitempty"`
	Opacity     float64         `json:"opacity,omitempty"`
	Width       float64         `json:"width,omitempty"`
	Height      float64         `json:"height,omitempty"`
	Position    *CanvasPosition `json:"position,omitempty"`
}

type EdgeStyleRules struct {
	Color            string  `json:"color,omitempty"`
	Width            float64 `json:"width,omitempty"`
	LineStyle        string  `json:"lineStyle,omitempty"`
	TargetArrowShape string  `json:"targetArrowShape,omitempty"`
	SourceArrowShape string  `json:"sourceArrowShape,omitempty"`
	Opacity          float64 `json:"opacity,omitempty"`
	LabelColor       string  `json:"labelColor,omitempty"`
}

type CanvasPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
