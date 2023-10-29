//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "GooseService=GooseService"
package external

type GooseService interface {
	DoSome() error
}
