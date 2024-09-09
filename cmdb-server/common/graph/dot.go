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

	//v, _ := json.Marshal(lines)
	//fmt.Printf("lines: \n%s\n", v)
	for _, line := range lines {
		dotLine := renderLine(line.Setting, line.DataList, line.MetaData, renderedItems)
		//fmt.Printf("%d ---> \n %s", index, dotLine)
		dot += dotLine
	}

	dot += "}\n"

	return
}

// renderChildren 渲染给定父元素的所有子元素
func renderChildren(children []*models.GraphElementNode, graphData MapData, meta MetaData) RenderResult {
	var dotString string
	var lines []Line
	var renderedItems []string

	for _, child := range children {
		ret := renderChild(child, graphData, meta)
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
	//newMeta := meta
	//newMeta := copyMetaData(meta)

	// todo
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

	var dotString strings.Builder

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

		if el.Children != nil {
			ret := renderChildren(el.Children, data, meta)
			lines = append(lines, ret.Lines...)
			renderedItems = append(renderedItems, ret.RenderedItems...)
			dotString.WriteString(ret.DotString)
		}

		dotString.WriteString("}\n")

	}
	return RenderResult{DotString: dotString.String(), Lines: lines, RenderedItems: renderedItems}
}

func renderImage(el *models.GraphElementNode, parentGuid string, dataList []MapData, meta MetaData) RenderResult {
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
	for _, data := range dataList {
		guid := mapGetStringAttr(data, "guid")
		// 检查过滤条件
		if isFilterFailed(el, data) {
			continue
		}

		// 检查是否已经渲染
		if isIn(guid, *meta.RenderedItems) {
			continue
		}
		renderedItems = append(renderedItems, guid)
		var nodeString strings.Builder

		// 根据图形类型构造节点字符串
		if meta.GraphType == "group" {
			nodeString.WriteString(fmt.Sprintf(`{rank=same;"%s"; %s`, el.NodeGroupName, guid))
		} else {
			nodeString.WriteString(guid)
		}

		// 渲染标签
		label := renderLabel(el.DisplayExpression, data)

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
		shape := getShape(el.GraphShapeData, el.GraphShapes, data, defaultShape)
		style := getStyle(el.GraphConfigData, el.GraphConfigs, data, meta, defaultImageStyle)

		// 扩展节点属性
		nodeAttrs = append(nodeAttrs, []string{
			fmt.Sprintf(`shape="%s"`, shape),
			`labelloc="b"`,
			fmt.Sprintf(`label="%s"`, calculateShapeLabel(shape, 1.1, meta.FontSize, label)),
			fmt.Sprintf(`image="%s"`, meta.ImagesMap[el.CiType]),
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
		if parentGuid != "" && meta.GraphType == "group" {
			dotString.WriteString(fmt.Sprintf(`%s -> %s [arrowsize=0]\n`, parentGuid, guid))
		}

		// 递归渲染子节点
		ret := renderChildren(el.Children, data, meta)
		dotString.WriteString(ret.DotString)
		lines = append(lines, ret.Lines...)
		renderedItems = append(renderedItems, ret.RenderedItems...)

		// 添加额外的连线信息
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
		DotString:     dotString.String(),
		Lines:         lines,
		RenderedItems: renderedItems,
		Error:         nil,
	}
}

func renderNode(el *models.GraphElementNode, parentGuid string, dataList []MapData, meta MetaData) RenderResult {
	var renderedItems []string
	var lines []Line
	var dotString strings.Builder
	defaultShape := "ellipse"

	// Iterate over each data in dataList
	for _, data := range dataList {
		guid := mapGetStringAttr(data, "guid")

		// If the node doesn't pass the filter, continue
		if isFilterFailed(el, data) {
			continue
		}

		// If the node is already rendered, skip it
		if isIn(guid, *meta.RenderedItems) {
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
		label := renderLabel(el.DisplayExpression, data)

		// Determine the node's shape
		shape := getShape(el.GraphShapeData, el.GraphShapes, data, defaultShape)

		// Calculate label size based on shape
		newLabel := calculateShapeLabel(shape, nodeWidth, meta.FontSize, label)

		// Get node style
		style := getStyle(el.GraphConfigData, el.GraphConfigs, data, meta, defaultStyle)

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
			dotString.WriteString(fmt.Sprintf("{rank=same;\"%s\"; %s}\n", el.NodeGroupName, nodeString))
		}

		// Recursively render children of the current node
		ret := renderChildren(el.Children, data, meta)
		dotString.WriteString(ret.DotString)
		lines = append(lines, ret.Lines...)
		renderedItems = append(renderedItems, ret.RenderedItems...)

		// If the node has line start and end data, generate line connections
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
		DotString:     dotString.String(),
		Lines:         lines,
		RenderedItems: renderedItems,
	}
}

func renderLine(el *models.GraphElementNode, dataList []MapData, meta MetaData, renderedItems []string) string {
	var dotString strings.Builder
	defaultShape := "normal"
	var lines []Line

	// Iterate through each data item in the dataList
	for _, data := range dataList {
		// If the filter fails for the current data, continue to the next item
		if isFilterFailed(el, data) {
			continue
		}

		// Iterate over el elements in the setting
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
				dotString.WriteString(ret.DotString)
				lines = append(lines, ret.Lines...)
				renderedItems = append(renderedItems, ret.RenderedItems...)
			}

		}

		// Get head and tail lines
		headLines := asList(data[el.LineStartData])
		tailLines := asList(data[el.LineEndData])

		// Generate line strings between head and tail
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
				dotString.WriteString(lineString)
			}
		}
	}

	return dotString.String()
}

func arrangeNodes(nodes []MapData) string {
	var dotString strings.Builder
	var rowHeadNodes []string

	if len(nodes) > 3 {
		// 使用 math.Ceil 进行向上取整
		numRow := int(math.Ceil(math.Sqrt(float64(len(nodes)))))

		// 遍历节点并生成 DOT 格式的字符串
		for index, node := range nodes {
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
		if (len(nodes)-1)%numRow != numRow-1 {
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
