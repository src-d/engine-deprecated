package engine

import (
	"context"
	"fmt"

	"database/sql"

	"github.com/docker/docker/api/types/container"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/src-d/engine/api"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
)

const (
	gitbasePort           = 3306
	gitbaseMountPath      = "/opt/repos"
	gitbaseIndexMountPath = "/var/lib/gitbase/index"
)

var (
	gitbase = components.Gitbase
)

func (s *Server) SQL(req *api.SQLRequest, stream api.Engine_SQLServer) error {
	err := s.startComponent(stream.Context(), gitbase.Name)
	if err != nil {
		return err
	}

	cfg := mysql.Config{
		User:                 "root",
		Net:                  "tcp",
		Addr:                 gitbase.Name,
		AllowNativePasswords: true,
		MaxAllowedPacket:     32 * (2 << 10),
	}
	logrus.Infof("connecting to mysql %q", cfg.FormatDSN())
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return errors.Wrap(err, "could not connect to gitbase")
	}
	rows, err := db.Query(req.Query)
	if err != nil {
		return errors.Wrap(err, "SQL query failed")
	}
	columns, err := rows.Columns()
	if err != nil {
		return errors.Wrap(err, "could not fetch columns")
	}

	columnsBytes := make([][]byte, len(columns))
	for i, c := range columns {
		columnsBytes[i] = []byte(c)
	}

	if err := stream.Send(&api.SQLResponse{
		Row: &api.SQLResponse_Row{Cell: columnsBytes},
	}); err != nil {
		return err
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new([]byte)
	}
	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			return errors.Wrap(err, "could not scan row")
		}
		row := &api.SQLResponse_Row{}
		for _, v := range values {
			row.Cell = append(row.Cell, *v.(*[]byte))
		}
		if err := stream.Send(&api.SQLResponse{
			Row: row,
		}); err != nil {
			return err
		}
	}

	return errors.Wrap(rows.Err(), "closing row iterator")
}

func createGitbase(opts ...docker.ConfigOption) docker.StartFunc {
	return func(ctx context.Context) error {
		if err := docker.EnsureInstalled(gitbase.Image, gitbase.Version); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), startComponentTimeout)
		defer cancel()

		config := &container.Config{
			Image: gitbase.ImageWithVersion(),
			Env: []string{
				fmt.Sprintf("BBLFSH_ENDPOINT=%s:%d", bblfshd.Name, bblfshParsePort),
			},
		}
		host := &container.HostConfig{}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, gitbase.Name)
	}
}
