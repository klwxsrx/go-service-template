//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Service=Service"
package goose

type Service interface {
	DoSome() error
}
