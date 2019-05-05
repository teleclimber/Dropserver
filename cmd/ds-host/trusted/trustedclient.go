package trusted

import (
	"net/rpc"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// RPCClient is used to communicate with ds-trusted
type RPCClient struct {
	client *rpc.Client
	Logger domain.LogCLientI
}

// Init makes the connection with the Trusted container
func (r *RPCClient) Init(ip string) {
	client, err := rpc.DialHTTP("tcp", ip+":1234") //preserve client in trusted struct so we can call again and stop it too
	if err != nil {
		r.Logger.Log(domain.ERROR, nil, "error dialing ds-trusted "+err.Error())
	}

	r.client = client
}

func (r *RPCClient) call(fnName string, args interface{}, reply interface{}) domain.Error {
	// reply has to be a pointer. Can we check?
	err := r.client.Call("TrustedAPI."+fnName, args, reply)
	if err != nil {
		if !dserror.Encoded(err) {
			// any error that is not an encoded ds-error happened on the rpc side, and is a loggable problem.
			r.Logger.Log(domain.ERROR, nil, "trustedclient: "+err.Error())
		}
		return dserror.FromStandard(err)
	}

	return nil
}

// SaveAppFiles saves files to ds-trusted
func (r *RPCClient) SaveAppFiles(args *domain.TrustedSaveAppFiles) (*domain.TrustedSaveAppFilesReply, domain.Error) {
	reply := domain.TrustedSaveAppFilesReply{}
	err := r.call("SaveAppFiles", args, &reply)

	// what kind of errors are we expecting here?
	// Could be a variety:
	// - ds-trusted could be down / unresponsive.
	// - rpc level things
	// - out of disk space
	// - upload too large, too many files, dir too deep, filenames too long, ...
	// - error in processing data into files
	// - error processing application files (app.json can't unmarshall, nonsensical data ...)
	//   ^^ this last one may be for another function like get App metadata

	return &reply, err
}

// GetAppMeta returns the app lication file metadata for the location key
func (r *RPCClient) GetAppMeta(args *domain.TrustedGetAppMeta) (*domain.TrustedGetAppMetaReply, domain.Error) {
	reply := domain.TrustedGetAppMetaReply{}
	err := r.call("GetAppMeta", args, &reply)

	return &reply, err
}
