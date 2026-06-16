package application

type ServiceProvider interface {
	CatalogReadService() *ReadService
	CatalogWriteService() *WriteService
}
