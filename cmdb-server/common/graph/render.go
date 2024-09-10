package graph

import (
	"github.com/WeBankPartners/we-cmdb/cmdb-server/models"
)

func Render(graph models.GraphQuery, dataList []map[string]interface{}, option RenderOption) (string, error) {
	//v, _ := json.Marshal(graph)
	//fmt.Printf("graph: \n%s\n", v)

	//v1, _ := json.Marshal(dataList)
	//fmt.Printf("data: \n%s\n", v1)
	//err := os.WriteFile("test.json", []byte(v), 0666)
	//if err != nil {
	//	log.Fatal(err)
	//}

	if graph.ViewGraphType == "sequence" {
		return RenderMermaid(graph, dataList, option)
	}
	return RenderDot(graph, dataList, option)
}
