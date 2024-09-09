package graph

import "github.com/WeBankPartners/we-cmdb/cmdb-server/models"

func RenderGraph(graph models.GraphQuery, dataList []MapData, option RenderOption) (string, error) {
	if graph.ViewGraphType == "sequence" {
		return RenderMermaid(graph, dataList, option)
	}
	return RenderDot(graph, dataList, option)
}
