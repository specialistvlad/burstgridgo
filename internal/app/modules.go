package app

import (
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/modules/env_vars"
	"github.com/vk/burstgridgo/modules/http_client"
	"github.com/vk/burstgridgo/modules/http_request"
	"github.com/vk/burstgridgo/modules/print"
	"github.com/vk/burstgridgo/modules/s3"
	"github.com/vk/burstgridgo/modules/socketio"
	"github.com/vk/burstgridgo/modules/socketio_client"
	"github.com/vk/burstgridgo/modules/socketio_request"
)

// coreModules is the definitive list of all modules that are compiled into
// the burstgridgo binary.
var coreModules = []registry.Module{
	&env_vars.Module{},
	&print.Module{},
	&http_request.Module{},
	&http_client.Module{},
	&s3.Module{},
	&socketio.Module{},
	&socketio_client.Module{},
	&socketio_request.Module{},
}
