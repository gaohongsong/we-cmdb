package graph

import (
	"encoding/json"
	"fmt"
	"github.com/WeBankPartners/we-cmdb/cmdb-server/models"
	"log"
	"math"
	"strconv"
	"strings"
)

var (
	defaultStyle = "penwidth=1;color=black;"
)

// RenderDot 渲染dot图
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
			RenderedItems: renderedItems,
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
	for _, line := range lines {
		dot += renderLine(line.Setting, line.DataList, line.MetaData, renderedItems)
	}

	dot += "}\n"

	return
}

// renderChildren 渲染给定父元素的所有子元素
func renderChildren(children []*models.GraphElementNode, graphElement MapData, meta MetaData) RenderResult {
	var dotString string
	var lines []Line
	var renderedItems []string

	for _, child := range children {
		ret := renderChild(child, graphElement, meta)
		dotString += ret.DotString
		lines = append(lines, ret.Lines...)
		renderedItems = append(renderedItems, ret.RenderedItems...)
	}

	return RenderResult{
		DotString:     dotString,
		Lines:         lines,
		RenderedItems: renderedItems,
	}
}

// renderChild 渲染子元素
func renderChild(child *models.GraphElementNode, graphData MapData, meta MetaData) (ret RenderResult) {
	// 子图字体逐级调小
	if meta.GraphType == "subgraph" {
		// 修改meta的副本不会影响外面的meta，不再需要手动copy
		meta.FontSize = math.Round((meta.FontSize-meta.FontStep)*100) / 100
	}

	// todo
	var childData []MapData
	tmp, _ := json.Marshal(graphData[child.DataName])
	err := json.Unmarshal(tmp, &childData)
	if err != nil {
		log.Fatal(err)
	}

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
		ret = renderImage(child, graphData, childData, meta)
		ret.DotString += nodeDot
	case "node":
		ret = renderNode(child, graphData, childData, meta)
		ret.DotString += nodeDot
	case "line":
		ret = RenderResult{
			Lines: []Line{{
				Setting:  child,
				DataList: childData,
				MetaData: meta,
			}},
		}
	default:
		panic("graph type not supported: " + child.GraphType)
	}
	return
}

func renderSubgraph(child *models.GraphElementNode, graphElements []MapData, meta MetaData) RenderResult {
	var renderedItems []string
	var lines []Line

	var dotString strings.Builder

	for _, gEl := range graphElements {
		if isFilterFailed(child, gEl) {
			continue
		}
		guid := mapGetStringAttr(gEl, "guid")
		keyName := mapGetStringAttr(gEl, "key_name")

		if isIn(guid, meta.RenderedItems) {
			continue
		}

		renderedItems = append(renderedItems, guid)

		label := renderLabel(child.DisplayExpression, gEl)
		style := getStyle(child.GraphConfigData, child.GraphConfigs, gEl, meta, defaultStyle)

		tooltip := keyName
		if tooltip == "" {
			tooltip = label
		}
		// 生成节点属性
		subgraphAttrs := []string{
			"id=" + guid,
			"fontsize=" + strconv.FormatFloat(meta.FontSize, 'f', -1, 64),
			"label=\"" + label + "\"",
			"tooltip=\"" + tooltip + "\"",
			style,
		}

		// 写入 subgraph
		dotString.WriteString(fmt.Sprintf("subgraph cluster_%s {\n", guid))
		dotString.WriteString(strings.Join(subgraphAttrs, ";") + "\n")
		dotString.WriteString(fmt.Sprintf("%s[penwidth=0;width=0;height=0;label=\"\"];\n", guid))

		if child.Children != nil {
			ret := renderChildren(child.Children, gEl, meta)
			lines = append(lines, ret.Lines...)
			renderedItems = append(renderedItems, ret.RenderedItems...)
			dotString.WriteString(ret.DotString)
		}

		dotString.WriteString("}\n")

	}
	return RenderResult{DotString: dotString.String(), Lines: lines, RenderedItems: renderedItems}
}

func renderImage(child *models.GraphElementNode, parent MapData, graphDataList []MapData, meta MetaData) RenderResult {
	var renderedItems []string
	var lines []Line
	var dotString strings.Builder
	var defaultImageStyle string
	defaultShape := "box"

	// 设置默认样式
	if meta.SuportVersion == "yes" {
		defaultImageStyle = `color="#dddddd";penwidth=1;`
	} else {
		defaultImageStyle = `color="transparent";penwidth=1;`
	}

	// 遍历节点数据
	parentGuid := mapGetStringAttr(parent, "guid")
	for _, data := range graphDataList {
		guid := mapGetStringAttr(data, "guid")
		// 检查过滤条件
		if isFilterFailed(child, data) {
			continue
		}

		// 检查是否已经渲染
		if isIn(guid, meta.RenderedItems) {
			continue
		}

		renderedItems = append(renderedItems, guid)
		var nodeString strings.Builder

		// 根据图形类型构造节点字符串
		if meta.GraphType == "group" {
			nodeString.WriteString(fmt.Sprintf(`{rank=same;"%s"; %s`, child.NodeGroupName, guid))
		} else {
			nodeString.WriteString(guid)
		}

		// 渲染标签
		label := renderLabel(child.DisplayExpression, data)

		// 节点属性
		nodeAttrs := []string{
			"id=" + guid,
			"fontsize=" + strconv.FormatFloat(meta.FontSize, 'g', 4, 64),
			"width=1.1",
			"height=1.1",
			fmt.Sprintf(`tooltip="%s"`, label),
			"fixedsize=true",
		}

		// 获取节点形状和样式
		shape := getShape(child.GraphShapeData, child.GraphShapes, data, defaultShape)
		style := getStyle(child.GraphConfigData, child.GraphConfigs, data, meta, defaultImageStyle)

		// 扩展节点属性
		nodeAttrs = append(nodeAttrs, []string{
			fmt.Sprintf(`shape="%s"`, shape),
			`labelloc="b"`,
			fmt.Sprintf(`label="%s"`, calculateShapeLabel(shape, 1.1, meta.FontSize, label)),
			fmt.Sprintf(`image="%s"`, meta.ImagesMap[child.CiType]),
			style,
		}...)

		// 生成节点 DOT 字符串
		nodeString.WriteString("[" + strings.Join(nodeAttrs, ";") + "]\n")

		// 如果是 group 类型，则闭合 rank
		if meta.GraphType == "group" {
			nodeString.WriteString("}")
		}

		// 追加到 dotString 中
		dotString.WriteString(nodeString.String())

		// 如果存在父节点并且图形类型是 group，则添加连线
		if parent != nil && meta.GraphType == "group" {
			dotString.WriteString(fmt.Sprintf(`%s -> %s [arrowsize=0]\n`, parentGuid, guid))
		}

		// 递归渲染子节点
		ret := renderChildren(child.Children, data, meta)
		dotString.WriteString(ret.DotString)
		lines = append(lines, ret.Lines...)
		renderedItems = append(renderedItems, ret.RenderedItems...)

		// 添加额外的连线信息
		if child.LineStartData != "" && child.LineEndData != "" {
			child.Children = nil
			lines = append(lines, Line{
				Setting:  child,
				MetaData: meta,
				DataList: []MapData{data},
			})
		}
	}

	return RenderResult{
		DotString:     dotString.String(),
		Lines:         lines,
		RenderedItems: renderedItems,
		Error:         nil,
	}
}

func renderNode(child *models.GraphElementNode, parent MapData, graphDataList []MapData, meta MetaData) RenderResult {
	var renderedItems []string
	var lines []Line
	var dotString strings.Builder
	defaultShape := "ellipse"

	// Iterate over each data in graphDataList
	for _, data := range graphDataList {
		guid := mapGetStringAttr(data, "guid")

		// If the node doesn't pass the filter, continue
		if isFilterFailed(child, data) {
			continue
		}

		// If the node is already rendered, skip it
		if isIn(guid, meta.RenderedItems) {
			continue
		}

		// Add this node's GUID to the list of rendered items
		renderedItems = append(renderedItems, guid)

		// Define node attributes
		nodeWidth := 4.0
		nodeAttrs := []string{
			fmt.Sprintf("id=%s", guid),
			fmt.Sprintf("fontsize=%s", strconv.FormatFloat(meta.FontSize, 'f', -1, 64)),
		}

		// Render the node's label
		label := renderLabel(child.DisplayExpression, data)

		// Determine the node's shape
		shape := getShape(child.GraphShapeData, child.GraphShapes, data, defaultShape)

		// Calculate label size based on shape
		newLabel := calculateShapeLabel(shape, nodeWidth, meta.FontSize, label)

		// Get node style
		style := getStyle(child.GraphConfigData, child.GraphConfigs, data, meta, defaultStyle)

		// Add shape, width, label, tooltip, and style to attributes
		nodeAttrs = append(nodeAttrs,
			fmt.Sprintf("shape=\"%s\"", shape),
			fmt.Sprintf("width=%.1f", nodeWidth),
			fmt.Sprintf("label=\"%s\"", newLabel),
			fmt.Sprintf("tooltip=\"%s\"", label),
			style,
		)

		// Add the node to the DOT string
		nodeString := fmt.Sprintf("%s[%s];\n", guid, strings.Join(nodeAttrs, ";"))
		dotString.WriteString(nodeString)

		// If this is a group graph type, add rank same logic
		if meta.GraphType == "group" {
			dotString.WriteString(fmt.Sprintf("{rank=same;\"%s\"; %s}\n", child.NodeGroupName, nodeString))
		}

		// Recursively render children of the current node
		ret := renderChildren(child.Children, data, meta)
		dotString.WriteString(ret.DotString)
		lines = append(lines, ret.Lines...)
		renderedItems = append(renderedItems, ret.RenderedItems...)

		// If the node has line start and end data, generate line connections
		if child.LineStartData != "" && child.LineEndData != "" {
			child.Children = nil
			//child.Children = []*models.GraphElementNode{}
			//child.Children = make([]*models.GraphElementNode, 0)
			lines = append(lines, Line{
				Setting:  child,
				MetaData: meta,
				DataList: []MapData{data},
			})
		}
	}

	return RenderResult{
		DotString:     dotString.String(),
		Lines:         lines,
		RenderedItems: renderedItems,
	}
}

func renderLine(settings *models.GraphElementNode, graphDataList []MapData, meta MetaData, renderedItems []string) string {
	var dotString strings.Builder
	defaultShape := "normal"
	var lines []Line

	// 子图字体逐级调小
	if meta.GraphType == "subgraph" {
		// 修改meta的副本不会影响外面的meta，不再需要手动copy
		meta.FontSize = math.Round((meta.FontSize-meta.FontStep)*100) / 100
	}

	// Iterate through each data item in the graphDataList
	for _, data := range graphDataList {
		// If the filter fails for the current data, continue to the next item
		if isFilterFailed(settings, data) {
			continue
		}

		// todo
		var childData []MapData
		tmp, _ := json.Marshal(data[settings.DataName])
		_ = json.Unmarshal(tmp, &childData)

		// Iterate over settings elements in the setting
		for _, child := range settings.Children {
			var ret RenderResult

			// Process based on the graphType
			switch child.GraphType {
			case "subgraph":
				ret = renderSubgraph(child, childData, meta)
			case "image":
				ret = renderImage(child, data, childData, meta)
			case "node":
				ret = renderNode(child, data, childData, meta)
			case "line":
				lines = append(lines, Line{
					Setting:  child,
					DataList: graphDataList,
					MetaData: meta,
				})
			}

			if child.GraphType != "line" {
				dotString.WriteString(ret.DotString)
				lines = append(lines, ret.Lines...)
				renderedItems = append(renderedItems, ret.RenderedItems...)
			}

		}

		// Get head and tail lines
		headLines := asList(data[settings.LineStartData])
		tailLines := asList(data[settings.LineEndData])

		// Generate line strings between head and tail
		for _, hLine := range headLines {
			for _, tLine := range tailLines {
				// Only render lines if both head and tail are already rendered
				//if !isIn(hLine, renderedItems) || !isIn(tLine, renderedItems) {
				//	fmt.Printf("ignore line: %s -> %s\n items=%s", hLine, tLine, renderedItems)
				//	continue
				//}

				// Create line string
				lineString := fmt.Sprintf("%s -> %s", hLine, tLine)
				lineAttrs := []string{
					fmt.Sprintf(`id="%s"`, data["guid"].(string)),
					fmt.Sprintf("fontsize=%.2f", meta.FontSize*0.6),
				}

				// Handle labels based on display position of line
				if settings.GraphType == "line" {
					label := renderLabel(settings.DisplayExpression, data)
					switch settings.LineDisplayPosition {
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
				if settings.GraphType == "line" {
					shape := getShape(settings.GraphShapeData, settings.GraphShapes, data, defaultShape)
					lineAttrs = append(lineAttrs, fmt.Sprintf("arrowhead=%s", shape))

					style := getStyle(settings.GraphConfigData, settings.GraphConfigs, data, meta, defaultStyle)
					lineAttrs = append(lineAttrs, style)
				} else {
					lineAttrs = append(lineAttrs, "arrowhead=icurve")
				}

				// Append the line to the dot string
				lineString += fmt.Sprintf("[%s];\n", strings.Join(lineAttrs, ";"))
				dotString.WriteString(lineString)
			}
		}
	}

	return dotString.String()
}

func arrangeNodes(graphElements []MapData) string {
	var dotString strings.Builder
	var rowHeadNodes []string

	if len(graphElements) > 3 {
		// 使用 math.Ceil 进行向上取整
		numRow := int(math.Ceil(math.Sqrt(float64(len(graphElements)))))

		// 遍历节点并生成 DOT 格式的字符串
		for index, node := range graphElements {
			guid := node["guid"].(string)

			// 每行的第一个节点
			if index%numRow == 0 {
				dotString.WriteString("{rank=same;")
				rowHeadNodes = append(rowHeadNodes, guid)
			}

			// 添加当前节点的 guid
			dotString.WriteString(guid + ";")

			// 每行的最后一个节点
			if index%numRow == numRow-1 {
				dotString.WriteString("}\n")
			}
		}

		// 如果最后一行未填满，关闭 rank=same 的块
		if (len(graphElements)-1)%numRow != numRow-1 {
			dotString.WriteString("}\n")
		}

		// 将每行的第一个节点连接起来
		for i := 0; i < len(rowHeadNodes)-1; i++ {
			dotString.WriteString(fmt.Sprintf("%s->%s[penwidth=0;minlen=1;arrowsize=0];\n", rowHeadNodes[i], rowHeadNodes[i+1]))
		}
	}

	return dotString.String()
}

func countDepth(graph models.GraphQuery) int {
	// 深度搜索
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
