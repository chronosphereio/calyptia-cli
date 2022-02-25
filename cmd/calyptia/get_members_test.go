package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/calyptia/api/types"
)

func Test_newCmdGetMembers(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdGetMembers(configWithMock(nil))
		cmd.SetOutput(got)

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "EMAIL NAME ROLES AGE\n", got.String())

		t.Run("with_ids", func(t *testing.T) {
			got.Reset()
			cmd.SetArgs([]string{"--show-ids"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, "ID EMAIL NAME ROLES MEMBER-ID AGE\n", got.String())
		})
	})

	t.Run("error", func(t *testing.T) {
		want := errors.New("internal error")
		cmd := newCmdGetMembers(configWithMock(&ClientMock{
			MembersFunc: func(ctx context.Context, projectID string, params types.MembersParams) ([]types.Membership, error) {
				return nil, want
			},
		}))
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		got := cmd.Execute()
		wantEq(t, want, got)
	})

	t.Run("ok", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		want := []types.Membership{{
			ID:        "member_id_1",
			Roles:     []types.MembershipRole{types.MembershipRoleCreator},
			CreatedAt: now.Add(time.Minute * -5),
			User: &types.User{
				ID:    "user_id_1",
				Email: "email_1",
				Name:  "name_1",
			},
		}, {
			ID:        "member_id_2",
			Roles:     []types.MembershipRole{types.MembershipRoleAdmin},
			CreatedAt: now.Add(time.Minute * -2),
			User: &types.User{
				ID:    "user_id_2",
				Email: "email_2",
				Name:  "name_2",
			},
		}}
		got := &bytes.Buffer{}
		cmd := newCmdGetMembers(configWithMock(&ClientMock{
			MembersFunc: func(ctx context.Context, projectID string, params types.MembersParams) ([]types.Membership, error) {
				wantNoEq(t, nil, params.Last)
				wantEq(t, uint64(2), *params.Last)
				return want, nil
			},
		}))
		cmd.SetOutput(got)
		cmd.SetArgs([]string{"--last", "2"})

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, ""+
			"EMAIL   NAME   ROLES   AGE\n"+
			"email_1 name_1 creator 5 minutes\n"+
			"email_2 name_2 admin   2 minutes\n", got.String())

		t.Run("show_ids", func(t *testing.T) {
			got.Reset()
			cmd.SetArgs([]string{"--show-ids"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, ""+
				"ID        EMAIL   NAME   ROLES   MEMBER-ID   AGE\n"+
				"user_id_1 email_1 name_1 creator member_id_1 5 minutes\n"+
				"user_id_2 email_2 name_2 admin   member_id_2 2 minutes\n", got.String())
		})

		t.Run("json", func(t *testing.T) {
			want, err := json.Marshal(want)
			wantEq(t, nil, err)

			got.Reset()
			cmd.SetArgs([]string{"--output-format=json"})

			err = cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, string(want)+"\n", got.String())
		})
	})
}
