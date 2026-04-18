package endtoendtesting

import (
	"net/http"
	"os"
	"testing"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/applicationtest"
	"github.com/downballot/downballot/internal/testutils"
	"github.com/stretchr/testify/require"
	"github.com/tekkamanendless/restapiclient"
)

func TestCampaign(t *testing.T) {
	testutils.Setup(t)

	ctx := t.Context()

	application := applicationtest.New(t, ctx)
	t.Cleanup(func() {
		application.Close()
	})

	masterClient := application.AuthenticatedClientMaster()

	adminName := "John Smith"
	adminUsername := "jsmith@example.com"
	adminPassword := "abc123"
	adminUserId := ""
	var adminClient *downballotapi.Client

	organizationName := "My Organization"
	organizationId := ""
	rootGroupId := ""

	user1Name := "User One"
	user1Username := "user1@example.com"
	user1Password := "abc123"
	user1Id := ""
	var user1Client *downballotapi.Client

	user2Name := "User Two"
	user2Username := "user2@example.com"
	user2Password := "abc123"
	user2Id := ""
	var user2Client *downballotapi.Client

	group1Name := "Stenning Woods"
	group1Filter := "::residential_address_development = 'Stenning Woods'"
	group1Id := ""

	group2Name := "Corner Ketch"
	group2Filter := "::residential_address_development = 'Corner Ketch'"
	group2Id := ""

	t.Log("Register the admin user.")
	{
		input := downballotapi.RegisterUserRequest{
			Name:     adminName,
			Username: adminUsername,
			Password: adminPassword,
		}
		var output downballotapi.RegisterUserResponse
		err := application.AuthenticatedClientMaster().Do(ctx, http.MethodPost, "/api/v1/user", input, &output)
		require.NoError(t, err)
		adminUserId = output.ID
		t.Logf("User ID: %s", adminUserId)
	}

	t.Log("Register user 1.")
	{
		input := downballotapi.RegisterUserRequest{
			Name:     user1Name,
			Username: user1Username,
			Password: user1Password,
		}
		var output downballotapi.RegisterUserResponse
		err := application.AuthenticatedClientMaster().Do(ctx, http.MethodPost, "/api/v1/user", input, &output)
		require.NoError(t, err)
		user1Id = output.ID
		t.Logf("User 1 ID: %s", user1Id)
	}

	t.Log("Register user 2.")
	{
		input := downballotapi.RegisterUserRequest{
			Name:     user2Name,
			Username: user2Username,
			Password: user2Password,
		}
		var output downballotapi.RegisterUserResponse
		err := application.AuthenticatedClientMaster().Do(ctx, http.MethodPost, "/api/v1/user", input, &output)
		require.NoError(t, err)
		user2Id = output.ID
		t.Logf("User 2 ID: %s", user2Id)
	}

	t.Log("Log in as the admin user.")
	{
		adminClient = application.UnauthenticatedClient()
		err := adminClient.Login(ctx, &downballotapi.LoginRequest{
			Username: adminUsername,
			Password: adminPassword,
		})
		require.NoError(t, err)
	}

	t.Log("Log in as user 1.")
	{
		user1Client = application.UnauthenticatedClient()
		err := user1Client.Login(ctx, &downballotapi.LoginRequest{
			Username: user1Username,
			Password: user1Password,
		})
		require.NoError(t, err)
	}

	t.Log("Log in as user 2.")
	{
		user2Client = application.UnauthenticatedClient()
		err := user2Client.Login(ctx, &downballotapi.LoginRequest{
			Username: user2Username,
			Password: user2Password,
		})
		require.NoError(t, err)
	}

	t.Log("Create a new organization as the admin user.")
	{
		input := downballotapi.RegisterOrganizationRequest{
			Name: organizationName,
		}
		var output downballotapi.RegisterOrganizationResponse
		err := adminClient.Do(ctx, http.MethodPost, "/api/v1/organization", input, &output)
		require.NoError(t, err)
		organizationId = output.ID
		t.Logf("Organization ID: %s", organizationId)
	}

	t.Log("Get the root group ID.")
	{
		var output downballotapi.GetGroupResponse
		err := adminClient.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/group/root", nil, &output)
		require.NoError(t, err)
		rootGroupId = output.Group.ID
		t.Logf("Root group ID: %s", rootGroupId)
	}

	t.Log("Add user 1 to the organization.")
	{
		input := downballotapi.AddUserToOrganizationRequest{
			Username: user1Username,
		}
		err := adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/user", input, nil)
		require.NoError(t, err)
	}

	t.Log("Add user 2 to the organization.")
	{
		input := downballotapi.AddUserToOrganizationRequest{
			Username: user2Username,
		}
		err := adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/user", input, nil)
		require.NoError(t, err)
	}

	t.Log("List the organizations as the master token.")
	{
		var output downballotapi.ListOrganizationsResponse
		err := masterClient.Do(ctx, http.MethodGet, "/api/v1/organization", nil, &output)
		require.NoError(t, err)
		t.Logf("Organizations: %v", output.Organizations)
	}

	t.Log("List the organizations as the admin user.")
	{
		var output downballotapi.ListOrganizationsResponse
		err := adminClient.Do(ctx, http.MethodGet, "/api/v1/organization", nil, &output)
		require.NoError(t, err)
		t.Logf("Organizations: %v", output.Organizations)
	}

	t.Log("List all persons as the admin user (there should be none).")
	{
		var output downballotapi.ListPersonsResponse
		err := adminClient.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/person", nil, &output)
		require.NoError(t, err)
		t.Logf("Persons: %v", output.Persons)
	}

	t.Log("Import the voter file as the admin user.")
	{
		input, err := os.ReadFile("../../test/de_voter_reg.small.csv")
		require.NoError(t, err)
		var output downballotapi.ImportPersonResponse
		err = adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/person/import", restapiclient.RawBytes(input), &output, restapiclient.OptionHeader("Content-Type", "application/octet-stream"))
		require.NoError(t, err)
		t.Logf("Persons: %v", output.Records)
	}

	t.Log("List all persons named Charls with 'whit' in the last name as the admin user.")
	{
		var output downballotapi.ListPersonsResponse
		err := adminClient.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/person?filter=::name_first+=+charles+AND+::name_last+~+whit", nil, &output)
		require.NoError(t, err)
		t.Logf("Persons: %v", output.Persons)
	}

	t.Logf("Create group 1 %q matching: %q", group1Name, group2Filter)
	{
		input := downballotapi.CreateGroupRequest{
			ParentID: rootGroupId,
			Name:     group1Name,
			Filter:   group1Filter,
		}
		var output downballotapi.CreateGroupResponse
		err := adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/group", input, &output)
		require.NoError(t, err)
		group1Id = output.ID
		t.Logf("Group 1 ID: %s", group1Id)
	}

	t.Logf("Create group 2 %q matching: %q", group2Name, group2Filter)
	{
		input := downballotapi.CreateGroupRequest{
			ParentID: rootGroupId,
			Name:     group2Name,
			Filter:   group2Filter,
		}
		var output downballotapi.CreateGroupResponse
		err := adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/group", input, &output)
		require.NoError(t, err)
		group2Id = output.ID
		t.Logf("Group 2 ID: %s", group2Id)
	}

	t.Log("Add user 1 to group 1.")
	{
		input := downballotapi.AddUserToGroupRequest{
			GroupID: group1Id,
		}
		err := adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/user/"+user1Id+"/group", input, nil)
		require.NoError(t, err)
	}

	t.Log("Add user 2 to group 1 as well.")
	{
		input := downballotapi.AddUserToGroupRequest{
			GroupID: group1Id,
		}
		err := adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/user/"+user2Id+"/group", input, nil)
		require.NoError(t, err)
	}

	t.Log("Add user 2 to group 2.")
	{
		input := downballotapi.AddUserToGroupRequest{
			GroupID: group1Id,
		}
		err := adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/user/"+user2Id+"/group", input, nil)
		require.NoError(t, err)
	}

	t.Log("List the persons as the admin user.")
	{
		var output downballotapi.ListPersonsResponse
		err := adminClient.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/person", nil, &output)
		require.NoError(t, err)

		t.Logf("Persons: %v", output.Persons)
	}

	t.Log("List the persons as user 1.")
	{
		var output downballotapi.ListPersonsResponse
		err := user1Client.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/person", nil, &output)
		require.NoError(t, err)

		t.Logf("Persons: %v", output.Persons)
	}

	t.Log("List the persons as user 2.")
	{
		var output downballotapi.ListPersonsResponse
		err := user2Client.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/person", nil, &output)
		require.NoError(t, err)

		t.Logf("Persons: %v", output.Persons)
	}
}
