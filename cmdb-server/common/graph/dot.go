package graph

import (
	"encoding/json"
	"fmt"
	"github.com/WeBankPartners/we-cmdb/cmdb-server/models"
	"math"
	"strconv"
	"strings"
)

var (
	defaultStyle = "penwidth=1;color=black;"
)

// RenderDot render dot graph
func RenderDot(graph models.GraphQuery, dataList []MapData, option RenderOption) (dot string, err error) {
	suportVersion := option.SuportVersion
	imageMap := option.ImageMap

	dot = "digraph G {\n"
	dot += fmt.Sprintf("rankdir=%s;edge[minlen=3];compound=true;\n", graph.GraphDir)
	if graph.ViewGraphType == "group" {
		dot += "Node [color=\"transparent\";fixedsize=\"true\";width=\"1.1\";height=\"1.1\";shape=box];\n"
		dot += "{\nnode [shape=plaintext];\n" + graph.NodeGroups + ";\n}\n"
	}

	var renderedItems []string
	var lines []Line

	for _, graphData := range dataList {
		confirmTime := mapGetStringAttr(graphData, "confirm_time")
		guid := mapGetStringAttr(graphData, "guid")
		keyName := mapGetStringAttr(graphData, "key_name")

		meta := MetaData{
			SuportVersion: suportVersion,
			GraphType:     graph.ViewGraphType,
			GraphDir:      graph.GraphDir,
			ConfirmTime:   confirmTime,
			FontSize:      14,
			FontStep:      0,
			ImagesMap:     imageMap,
			RenderedItems: &renderedItems,
		}
		label := renderLabel(graph.RootData.DisplayExpression, graphData)
		renderedItems = append(renderedItems, guid)

		tooltip := keyName
		if tooltip == "" {
			tooltip = label
		}

		switch graph.ViewGraphType {
		case "group":
			dot += fmt.Sprintf("{rank=same; \"%s\"; %s[id=\"%s\";label=\"%s\"; fontsize=%s; penwidth=1;width=2; image=\"%s\"; labelloc=\"b\"; shape=\"box\";",
				graph.RootData.NodeGroupName, guid, guid, label, strconv.FormatFloat(meta.FontSize, 'g', -1, 64), option.ImageMap[graph.RootData.CiType])

			if meta.SuportVersion == "yes" {
				dot += "color=\"#dddddd\";penwidth=1;"
			}

			dot += "}\n"

		case "subgraph":
			depth := countDepth(graph)
			meta.FontSize = 20
			meta.FontStep = ((meta.FontSize - 14.0) * 1.0) / float64(depth-1)
			dot += fmt.Sprintf("subgraph cluster_%s { \n", guid)
			dot += fmt.Sprintf("id=%s;\n", guid)
			dot += fmt.Sprintf("fontsize=%s;\n", strconv.FormatFloat(meta.FontSize, 'f', -1, 64))
			dot += fmt.Sprintf("label=\"%s\";\n", label)
			dot += fmt.Sprintf("tooltip=\"%s\";\n", tooltip)
			dot += fmt.Sprintf("%s[penwidth=0;width=0;height=0;label=\"\"];\n", guid)

			style := getStyle(graph.RootData.GraphConfigData, graph.RootData.GraphConfigs, graphData, meta, defaultStyle)
			dot += fmt.Sprintf("%s\n", style)
		}

		for _, child := range graph.RootData.Children {
			if isFilterFailed(child, graphData) {
				continue
			}

			ret := renderChild(child, graphData, meta)
			if ret.Error != nil {
				return "", ret.Error
			}
			dot += ret.DotString
			renderedItems = append(renderedItems, ret.RenderedItems...)
			lines = append(lines, ret.Lines...)
		}

		if graph.ViewGraphType == "subgraph" {
			dot += "}\n"
		}
	}

	v, _ := json.Marshal(lines)
	fmt.Printf("lines: \n%s\n", v)
	for index, line := range lines {
		dotLine := renderLine(line.Setting, line.DataList, line.MetaData, renderedItems)
		fmt.Printf("%d ---> \n %s", index, dotLine)
		dot += dotLine
	}

	dot += "}\n"

	return
}

// renderChildren render children elements
func renderChildren(children []*models.GraphElementNode, graphData MapData, meta MetaData) RenderResult {
	var dot string
	var lines []Line
	var renderedItems []string

	for _, child := range children {
		ret := renderChild(child, graphData, meta)
		dot += ret.DotString
		lines = append(lines, ret.Lines...)
		renderedItems = append(renderedItems, ret.RenderedItems...)
	}

	return RenderResult{
		DotString:     dot,
		Lines:         lines,
		RenderedItems: renderedItems,
	}
}

// renderChild render child element
func renderChild(child *models.GraphElementNode, graphData MapData, meta MetaData) (ret RenderResult) {
	if meta.GraphType == "subgraph" {
		meta.FontSize = math.Round((meta.FontSize-meta.FontStep)*100) / 100
	}
	//newMeta := meta
	//newMeta := copyMetaData(meta)

	var childData []MapData
	tmp, _ := json.Marshal(graphData[child.DataName])
	_ = json.Unmarshal(tmp, &childData)

	//fmt.Printf("childData: %v+", childData)
	//childData, ok = graphData[child.DataName].([]MapData)

	nodeDot := ""
	if child.GraphType != "subgraph" && meta.GraphType == "subgraph" {
		nodeDot = arrangeNodes(childData)
	}

	switch child.GraphType {
	case "subgraph":
		ret = renderSubgraph(child, childData, meta)
	case "image":
		parentGuid := mapGetStringAttr(graphData, "guid")
		ret = renderImage(child, parentGuid, childData, meta)
		ret.DotString += nodeDot
	case "node":
		parentGuid := mapGetStringAttr(graphData, "guid")
		ret = renderNode(child, parentGuid, childData, meta)
		ret.DotString += nodeDot
	case "line":
		ret = RenderResult{
			Lines: []Line{{
				Setting:  child,
				DataList: childData,
				MetaData: meta,
			}},
		}
	}
	return
}

func renderSubgraph(el *models.GraphElementNode, dataList []MapData, meta MetaData) RenderResult {
	var renderedItems []string
	var lines []Line

	var dot strings.Builder

	for _, data := range dataList {
		if isFilterFailed(el, data) {
			continue
		}
		guid := mapGetStringAttr(data, "guid")
		keyName := mapGetStringAttr(data, "key_name")

		if isIn(guid, *meta.RenderedItems) {
			continue
		}

		renderedItems = append(renderedItems, guid)

		label := renderLabel(el.DisplayExpression, data)
		style := getStyle(el.GraphConfigData, el.GraphConfigs, data, meta, defaultStyle)

		tooltip := keyName
		if tooltip == "" {
			tooltip = label
		}
		subgraphAttrs := []string{
			"id=" + guid,
			"fontsize=" + strconv.FormatFloat(meta.FontSize, 'f', -1, 64),
			"label=\"" + label + "\"",
			"tooltip=\"" + tooltip + "\"",
			style,
		}

		// 写入 subgraph
		dot.WriteString(fmt.Sprintf("subgraph cluster_%s {\n", guid))
		dot.WriteString(strings.Join(subgraphAttrs, ";") + "\n")
		dot.WriteString(fmt.Sprintf("%s[penwidth=0;width=0;height=0;label=\"\"];\n", guid))

		if el.Children != nil {
			ret := renderChildren(el.Children, data, meta)
			lines = append(lines, ret.Lines...)
			renderedItems = append(renderedItems, ret.RenderedItems...)
			dot.WriteString(ret.DotString)
		}

		dot.WriteString("}\n")

	}
	return RenderResult{DotString: dot.String(), Lines: lines, RenderedItems: renderedItems}
}

func renderImage(el *models.GraphElementNode, parentGuid string, dataList []MapData, meta MetaData) RenderResult {
	var renderedItems []string
	var lines []Line
	var dot strings.Builder
	var defaultImageStyle string
	defaultShape := "box"

	if meta.SuportVersion == "yes" {
		defaultImageStyle = `color="#dddddd";penwidth=1;`
	} else {
		defaultImageStyle = `color="transparent";penwidth=1;`
	}

	for _, data := range dataList {
		guid := mapGetStringAttr(data, "guid")
		if isFilterFailed(el, data) {
			continue
		}

		if isIn(guid, *meta.RenderedItems) {
			continue
		}
		renderedItems = append(renderedItems, guid)
		var nodeString strings.Builder

		if meta.GraphType == "group" {
			nodeString.WriteString(fmt.Sprintf(`{rank=same;"%s"; %s`, el.NodeGroupName, guid))
		} else {
			nodeString.WriteString(guid)
		}

		label := renderLabel(el.DisplayExpression, data)

		nodeAttrs := []string{
			"id=" + guid,
			"fontsize=" + strconv.FormatFloat(meta.FontSize, 'g', 4, 64),
			"width=1.1",
			"height=1.1",
			fmt.Sprintf(`tooltip="%s"`, label),
			"fixedsize=true",
		}

		shape := getShape(el.GraphShapeData, el.GraphShapes, data, defaultShape)
		style := getStyle(el.GraphConfigData, el.GraphConfigs, data, meta, defaultImageStyle)

		nodeAttrs = append(nodeAttrs, []string{
			fmt.Sprintf(`shape="%s"`, shape),
			`labelloc="b"`,
			fmt.Sprintf(`label="%s"`, calculateShapeLabel(shape, 1.1, meta.FontSize, label)),
			fmt.Sprintf(`image="%s"`, meta.ImagesMap[el.CiType]),
			style,
		}...)

		nodeString.WriteString("[" + strings.Join(nodeAttrs, ";") + "]\n")

		if meta.GraphType == "group" {
			nodeString.WriteString("}")
		}

		dot.WriteString(nodeString.String())

		if parentGuid != "" && meta.GraphType == "group" {
			dot.WriteString(fmt.Sprintf(`%s -> %s [arrowsize=0]\n`, parentGuid, guid))
		}

		ret := renderChildren(el.Children, data, meta)
		dot.WriteString(ret.DotString)
		lines = append(lines, ret.Lines...)
		renderedItems = append(renderedItems, ret.RenderedItems...)

		if el.LineStartData != "" && el.LineEndData != "" {
			el.Children = nil
			lines = append(lines, Line{
				Setting:  el,
				MetaData: meta,
				DataList: []MapData{data},
			})
		}
	}

	return RenderResult{
		DotString:     dot.String(),
		Lines:         lines,
		RenderedItems: renderedItems,
		Error:         nil,
	}
}

func renderNode(el *models.GraphElementNode, parentGuid string, dataList []MapData, meta MetaData) RenderResult {
	var renderedItems []string
	var lines []Line
	var dot strings.Builder
	defaultShape := "ellipse"

	for _, data := range dataList {
		guid := mapGetStringAttr(data, "guid")

		if isFilterFailed(el, data) {
			continue
		}

		if isIn(guid, *meta.RenderedItems) {
			continue
		}

		renderedItems = append(renderedItems, guid)

		nodeWidth := 4.0
		nodeAttrs := []string{
			fmt.Sprintf("id=%s", guid),
			fmt.Sprintf("fontsize=%s", strconv.FormatFloat(meta.FontSize, 'f', -1, 64)),
		}

		label := renderLabel(el.DisplayExpression, data)

		shape := getShape(el.GraphShapeData, el.GraphShapes, data, defaultShape)

		newLabel := calculateShapeLabel(shape, nodeWidth, meta.FontSize, label)

		style := getStyle(el.GraphConfigData, el.GraphConfigs, data, meta, defaultStyle)

		nodeAttrs = append(nodeAttrs,
			fmt.Sprintf("shape=\"%s\"", shape),
			fmt.Sprintf("width=%.1f", nodeWidth),
			fmt.Sprintf("label=\"%s\"", newLabel),
			fmt.Sprintf("tooltip=\"%s\"", label),
			style,
		)

		nodeString := fmt.Sprintf("%s[%s];\n", guid, strings.Join(nodeAttrs, ";"))
		dot.WriteString(nodeString)

		if meta.GraphType == "group" {
			dot.WriteString(fmt.Sprintf("{rank=same;\"%s\"; %s}\n", el.NodeGroupName, nodeString))
		}

		ret := renderChildren(el.Children, data, meta)
		dot.WriteString(ret.DotString)
		lines = append(lines, ret.Lines...)
		renderedItems = append(renderedItems, ret.RenderedItems...)

		if el.LineStartData != "" && el.LineEndData != "" {
			el.Children = nil
			//el.Children = []*models.GraphElementNode{}
			//el.Children = make([]*models.GraphElementNode, 0)
			lines = append(lines, Line{
				Setting:  el,
				MetaData: meta,
				DataList: []MapData{data},
			})
		}
	}

	return RenderResult{
		DotString:     dot.String(),
		Lines:         lines,
		RenderedItems: renderedItems,
	}
}

func renderLine(el *models.GraphElementNode, dataList []MapData, meta MetaData, renderedItems []string) string {
	var dot strings.Builder
	defaultShape := "normal"
	var lines []Line

	for _, data := range dataList {
		if isFilterFailed(el, data) {
			continue
		}

		for _, child := range el.Children {
			var ret RenderResult

			// todo
			var childData []MapData
			tmp, _ := json.Marshal(data[child.DataName])
			_ = json.Unmarshal(tmp, &childData)

			// 子图字体逐级调小
			if meta.GraphType == "subgraph" {
				// 修改meta的副本不会影响外面的meta，不再需要手动copy
				meta.FontSize = math.Round((meta.FontSize-meta.FontStep)*100) / 100
			}
			//newMeta := meta
			//newMeta := copyMetaData(meta)

			// Process based on the graphType
			switch child.GraphType {
			case "subgraph":
				ret = renderSubgraph(child, childData, meta)
			case "image":
				parentGuid := mapGetStringAttr(data, "guid")
				ret = renderImage(child, parentGuid, childData, meta)
			case "node":
				parentGuid := mapGetStringAttr(data, "guid")
				ret = renderNode(child, parentGuid, childData, meta)
			case "line":
				lines = append(lines, Line{
					Setting:  child,
					DataList: dataList,
					MetaData: meta,
				})
			}
			if child.GraphType != "line" {
				dot.WriteString(ret.DotString)
				lines = append(lines, ret.Lines...)
				renderedItems = append(renderedItems, ret.RenderedItems...)
			}

		}

		headLines := asList(data[el.LineStartData])
		tailLines := asList(data[el.LineEndData])

		for _, hLine := range headLines {
			for _, tLine := range tailLines {
				// Only render lines if both head and tail are already rendered
				if !isIn(hLine, renderedItems) || !isIn(tLine, renderedItems) {
					fmt.Printf("ignore line: %s -> %s\n items=%s", hLine, tLine, renderedItems)
					continue
				}

				// Create line string
				lineString := fmt.Sprintf("%s -> %s", hLine, tLine)
				lineAttrs := []string{
					fmt.Sprintf(`id="%s"`, data["guid"].(string)),
					fmt.Sprintf("fontsize=%.2f", meta.FontSize*0.6),
				}

				// Handle labels based on display position of line
				if el.GraphType == "line" {
					label := renderLabel(el.DisplayExpression, data)
					switch el.LineDisplayPosition {
					case "middle":
						lineAttrs = append(lineAttrs, fmt.Sprintf(`label="%s"`, label))
					case "head":
						lineAttrs = append(lineAttrs, fmt.Sprintf(`headlabel="%s"`, label))
					case "tail":
						lineAttrs = append(lineAttrs, fmt.Sprintf(`taillabel="%s"`, label))
					}
					lineAttrs = append(lineAttrs, fmt.Sprintf(`tooltip="%s"`, data["key_name"].(string)))
				}

				// Add attributes for clusters
				lineAttrs = append(lineAttrs, fmt.Sprintf("lhead=cluster_%s", tLine))
				lineAttrs = append(lineAttrs, fmt.Sprintf("ltail=cluster_%s", hLine))

				// Add arrowhead and style if it's a line graph type
				if el.GraphType == "line" {
					shape := getShape(el.GraphShapeData, el.GraphShapes, data, defaultShape)
					lineAttrs = append(lineAttrs, fmt.Sprintf("arrowhead=%s", shape))

					style := getStyle(el.GraphConfigData, el.GraphConfigs, data, meta, defaultStyle)
					lineAttrs = append(lineAttrs, style)
				} else {
					lineAttrs = append(lineAttrs, "arrowhead=icurve")
				}

				// Append the line to the dot string
				lineString += fmt.Sprintf("[%s];\n", strings.Join(lineAttrs, ";"))
				dot.WriteString(lineString)
			}
		}
	}

	return dot.String()
}

func arrangeNodes(nodes []MapData) string {
	var dot strings.Builder
	var rowHeadNodes []string

	if len(nodes) > 3 {
		numRow := int(math.Ceil(math.Sqrt(float64(len(nodes)))))

		for index, node := range nodes {
			guid := node["guid"].(string)

			if index%numRow == 0 {
				dot.WriteString("{rank=same;")
				rowHeadNodes = append(rowHeadNodes, guid)
			}

			dot.WriteString(guid + ";")

			if index%numRow == numRow-1 {
				dot.WriteString("}\n")
			}
		}

		if (len(nodes)-1)%numRow != numRow-1 {
			dot.WriteString("}\n")
		}

		for i := 0; i < len(rowHeadNodes)-1; i++ {
			dot.WriteString(fmt.Sprintf("%s->%s[penwidth=0;minlen=1;arrowsize=0];\n", rowHeadNodes[i], rowHeadNodes[i+1]))
		}
	}

	return dot.String()
}

func countDepth(graph models.GraphQuery) int {
	var dfs func(*models.GraphElementNode, int) int
	dfs = func(curNode *models.GraphElementNode, curDepth int) int {
		if curNode.Children == nil || len(curNode.Children) == 0 {
			return curDepth
		}

		maxDepth := curDepth
		for _, child := range curNode.Children {
			childDepth := dfs(child, curDepth+1)
			if childDepth > maxDepth {
				maxDepth = childDepth
			}
		}
		return maxDepth
	}

	maxDepth := 0
	for _, child := range graph.RootData.Children {
		childDepth := dfs(child, 1)
		if childDepth > maxDepth {
			maxDepth = childDepth
		}
	}
	return maxDepth
}

func copyMetaData(metaData MetaData) MetaData {
	newMetaData := MetaData{
		ConfirmTime:   metaData.ConfirmTime,
		FontSize:      metaData.FontSize,
		FontStep:      metaData.FontStep,
		GraphDir:      metaData.GraphDir,
		GraphType:     metaData.GraphType,
		ImagesMap:     metaData.ImagesMap,
		RenderedItems: metaData.RenderedItems,
		SuportVersion: metaData.SuportVersion,
	}

	if newMetaData.GraphType == "subgraph" {
		// 修改meta的副本不会影响外面的meta，不再需要手动copy
		newMetaData.FontSize = math.Round((newMetaData.FontSize-newMetaData.FontStep)*100) / 100
	}

	return newMetaData
}
