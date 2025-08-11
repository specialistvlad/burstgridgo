package app

import (
	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/modules/env_vars"
	"github.com/specialistvlad/burstgridgo/modules/http_client"
	"github.com/specialistvlad/burstgridgo/modules/print"
	"github.com/specialistvlad/burstgridgo/modules/s3"
	"github.com/specialistvlad/burstgridgo/modules/socketio"
)

// coreModules is the definitive list of all modules that are compiled into
// the burstgridgo binary.
var coreModules = []registry.Module{
	&env_vars.Module{},
	&print.Module{},
	&http_client.Module{},
	&s3.Module{},
	&socketio.Module{},
}
