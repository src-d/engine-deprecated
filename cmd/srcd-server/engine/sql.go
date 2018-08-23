package engine

import (
	"context"
	"fmt"
	"time"

	"database/sql"

	"github.com/docker/docker/api/types/container"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/docker"
)

const (
	gitbaseName      = "srcd-cli-gitbase"
	gitbaseMountPath = "/opt/repos"
	gitbaseImage     = "srcd/gitbase"
)

func (s *Server) SQL(ctx context.Context, req *api.SQLRequest) (*api.SQLResponse, error) {

	err := Run(Component{
		Name: gitbaseName,
		Start: createGitbase(
			docker.WithVolume(s.workdir, gitbaseMountPath),
		),
		Dependencies: []Component{{
			Name:  bblfshdName,
			Start: createBbblfshd,
		}},
	})
	if err != nil {
		return nil, err
	}

	cfg := mysql.Config{
		User:                 "root",
		Net:                  "tcp",
		Addr:                 gitbaseName,
		AllowNativePasswords: true,
		MaxAllowedPacket:     32 * (2 << 10),
	}
	logrus.Infof("connecting to mysql %q", cfg.FormatDSN())
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to gitbase")
	}
	rows, err := db.Query(req.Query)
	if err != nil {
		return nil, errors.Wrap(err, "SQL query failed")
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch columns")
	}
	res := &api.SQLResponse{
		Header: &api.SQLResponse_Row{Cell: columns},
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(string)
	}
	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			return nil, errors.Wrap(err, "could not scan row")
		}
		row := &api.SQLResponse_Row{}
		for _, v := range values {
			row.Cell = append(row.Cell, *v.(*string))
		}
		res.Rows = append(res.Rows, row)
	}

	return res, errors.Wrap(rows.Err(), "closing row iterator")
}

func createGitbase(opts ...docker.ConfigOption) docker.StartFunc {
	return func() error {
		if err := docker.EnsureInstalled(gitbaseImage, ""); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		config := &container.Config{
			Image: gitbaseImage,
			Env:   []string{fmt.Sprintf("BBLFSH_ENDPOINT=%s:9432", bblfshdName)},
		}
		host := &container.HostConfig{}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, gitbaseName)
	}
}
