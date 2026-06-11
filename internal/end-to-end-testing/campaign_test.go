package endtoendtesting

import (
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/applicationtest"
	"github.com/downballot/downballot/internal/schema"
	"github.com/downballot/downballot/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tekkamanendless/go-mailer"
	"github.com/tekkamanendless/restapiclient"
)

func TestCampaign(t *testing.T) {
	testutils.Setup(t)

	ctx := t.Context()
	ctx = mailer.WithDummyMode(ctx)

	application := applicationtest.New(t, ctx)
	t.Cleanup(func() {
		application.Close()
	})

	masterClient := application.AuthenticatedClientMaster()

	adminName := "John Smith"
	adminUsername := "jsmith@example.com"
	adminUserId := ""
	var adminClient *downballotapi.Client

	organizationName := "My Organization"
	organizationId := ""
	rootGroupId := ""

	user1Name := "User One"
	user1Username := "user1@example.com"
	user1Id := ""
	var user1Client *downballotapi.Client

	user2Name := "User Two"
	user2Username := "user2@example.com"
	user2Id := ""
	var user2Client *downballotapi.Client

	group1Name := "Monkey family"
	group1Filter := "name_last = 'Monkey'"
	group1Id := ""

	group2Name := "Navy dogs"
	group2Filter := "political_party = 'Navy'"
	group2Id := ""

	t.Log("Register the admin user.")
	{
		input := downballotapi.RegisterUserRequest{
			Name:     adminName,
			Username: adminUsername,
		}
		var output downballotapi.RegisterUserResponse
		err := application.UnauthenticatedClient().Do(ctx, http.MethodPost, "/api/v1/user", input, &output)
		require.NoError(t, err)
		adminUserId = output.ID
		t.Logf("User ID: %s", adminUserId)
	}

	t.Log("Register user 1.")
	{
		input := downballotapi.RegisterUserRequest{
			Name:     user1Name,
			Username: user1Username,
		}
		var output downballotapi.RegisterUserResponse
		err := application.UnauthenticatedClient().Do(ctx, http.MethodPost, "/api/v1/user", input, &output)
		require.NoError(t, err)
		user1Id = output.ID
		t.Logf("User 1 ID: %s", user1Id)
	}

	t.Log("Register user 2.")
	{
		input := downballotapi.RegisterUserRequest{
			Name:     user2Name,
			Username: user2Username,
		}
		var output downballotapi.RegisterUserResponse
		err := application.UnauthenticatedClient().Do(ctx, http.MethodPost, "/api/v1/user", input, &output)
		require.NoError(t, err)
		user2Id = output.ID
		t.Logf("User 2 ID: %s", user2Id)
	}

	t.Log("Log in as the admin user.")
	{
		adminClient = application.UnauthenticatedClient()

		err := adminClient.Do(ctx, http.MethodPost, "/api/v1/authentication/email", downballotapi.EmailRequest{
			Email: adminUsername,
		}, nil)
		require.NoError(t, err)

		passwordRegexp := regexp.MustCompile(`(?m)^\s*(\d{6})`)

		message := mailer.Dummy().LastMessageInInbox(application.Config().SendGridAPIKey, adminUsername)
		require.NotNil(t, message)
		t.Logf("Message: %s", message.BodyPlainText)
		require.Equal(t, adminUsername, message.To.Address)
		require.Equal(t, "Your Downballot one-time password", message.Subject)
		matches := passwordRegexp.FindStringSubmatch(message.BodyPlainText)
		require.Len(t, matches, 2)
		adminPassword := matches[1]

		err = adminClient.Login(ctx, &downballotapi.LoginRequest{
			Username: adminUsername,
			Password: adminPassword,
		})
		require.NoError(t, err)
	}

	t.Log("Log in as user 1.")
	{
		user1Client = application.UnauthenticatedClient()

		err := user1Client.Do(ctx, http.MethodPost, "/api/v1/authentication/email", downballotapi.EmailRequest{
			Email: user1Username,
		}, nil)
		require.NoError(t, err)

		passwordRegexp := regexp.MustCompile(`(?m)^\s*(\d{6})`)

		message := mailer.Dummy().LastMessageInInbox(application.Config().SendGridAPIKey, user1Username)
		require.NotNil(t, message)
		t.Logf("Message: %s", message.BodyPlainText)
		require.Equal(t, user1Username, message.To.Address)
		require.Equal(t, "Your Downballot one-time password", message.Subject)
		matches := passwordRegexp.FindStringSubmatch(message.BodyPlainText)
		require.Len(t, matches, 2)
		user1Password := matches[1]

		err = user1Client.Login(ctx, &downballotapi.LoginRequest{
			Username: user1Username,
			Password: user1Password,
		})
		require.NoError(t, err)
	}

	t.Log("Log in as user 2.")
	{
		user2Client = application.UnauthenticatedClient()

		err := user2Client.Do(ctx, http.MethodPost, "/api/v1/authentication/email", downballotapi.EmailRequest{
			Email: user2Username,
		}, nil)
		require.NoError(t, err)

		passwordRegexp := regexp.MustCompile(`(?m)^\s*(\d{6})`)

		message := mailer.Dummy().LastMessageInInbox(application.Config().SendGridAPIKey, user2Username)
		require.NotNil(t, message)
		t.Logf("Message: %s", message.BodyPlainText)
		require.Equal(t, user2Username, message.To.Address)
		require.Equal(t, "Your Downballot one-time password", message.Subject)
		matches := passwordRegexp.FindStringSubmatch(message.BodyPlainText)
		require.Len(t, matches, 2)
		user2Password := matches[1]

		err = user2Client.Login(ctx, &downballotapi.LoginRequest{
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
		var output downballotapi.ListGroupsResponse
		err := adminClient.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/group?parent_id=null", nil, &output)
		require.NoError(t, err)
		require.Len(t, output.Groups, 1)
		rootGroupId = output.Groups[0].ID
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
		assert.Empty(t, output.Persons)
	}

	t.Log("Create the fields.")
	{
		fields := []string{
			"birthday_year",
			"candidate.connected",
			"candidate.date_called",
			"candidate.date_texted",
			"candidate.do_not_contact",
			"candidate.donated",
			"candidate.notes",
			"candidate.support",
			"coordinates",
			"county",
			"district_representative",
			"district_school",
			"district_senate",
			"name",
			"name_first",
			"name_middle",
			"name_last",
			"name_suffix",
			"phone_number",
			"political_party",
			"residential_address",
			"residential_address_development",
			"mailing_address",
			"voter_id",
			"voting_history",
		}
		for _, field := range fields {
			fieldType := schema.PersonFieldDefinitionTypeString
			allowEmpty := true
			allowedValues := []string{}
			allowedRegex := ""
			switch field {
			case "birthday_year":
				fieldType = schema.PersonFieldDefinitionTypeInteger
			case "candidate.connected":
				fieldType = schema.PersonFieldDefinitionTypeBoolean
				allowEmpty = false
			case "candidate.date_called":
				fieldType = schema.PersonFieldDefinitionTypeDate
				allowEmpty = false
			case "candidate.date_texted":
				fieldType = schema.PersonFieldDefinitionTypeDate
				allowEmpty = false
			case "candidate.do_not_contact":
				fieldType = schema.PersonFieldDefinitionTypeBoolean
				allowEmpty = false
			case "candidate.donated":
				fieldType = schema.PersonFieldDefinitionTypeBoolean
			case "candidate.notes":
				fieldType = schema.PersonFieldDefinitionTypeString
			case "candidate.support":
				fieldType = schema.PersonFieldDefinitionTypeEnum
				allowedValues = []string{"-2", "-1", "0", "+1", "+2"}
			case "coordinates":
				fieldType = schema.PersonFieldDefinitionTypeCoordinates
			case "district_representative":
				allowEmpty = false
			case "district_school":
				allowEmpty = false
			case "district_senate":
				allowEmpty = false
			case "name":
				allowEmpty = false
			case "residential_address":
				allowEmpty = false
			case "voting_history":
				fieldType = schema.PersonFieldDefinitionTypeSet
			}
			err := adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/person-field", downballotapi.CreatePersonFieldRequest{
				Name:          field,
				Type:          downballotapi.PersonFieldDefinitionType(fieldType),
				AllowEmpty:    allowEmpty,
				AllowedValues: allowedValues,
				AllowedRegex:  allowedRegex,
			}, nil)
			require.NoError(t, err)
		}
	}

	t.Log("Import the voter file as the admin user.")
	{
		input, err := os.ReadFile("testdata/voterfile.csv")
		require.NoError(t, err)
		var output downballotapi.ImportPersonResponse
		err = adminClient.Do(ctx, http.MethodPost, "/api/v1/organization/"+organizationId+"/person/import", restapiclient.RawBytes(input), &output, restapiclient.OptionHeader("Content-Type", "text/csv"))
		require.NoError(t, err)
		t.Logf("Persons: %v", output.Records)
	}

	t.Log("List all persons named Garp with 'monk' in the last name as the admin user.")
	{
		var output downballotapi.ListPersonsResponse
		err := adminClient.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/person?filter=name_first+=+garp+AND+name_last+~+'*monk*'", nil, &output)
		require.NoError(t, err)
		t.Logf("Persons: %v", output.Persons)
		assert.Len(t, output.Persons, 1)
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
			GroupID: group2Id,
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
		assert.Len(t, output.Persons, 11)
		names := []string{}
		for _, person := range output.Persons {
			names = append(names, person.Fields["name"])
		}
		assert.Contains(t, names, "LUFFY D MONKEY")
		assert.Contains(t, names, "ZORO RORONOA")
		assert.Contains(t, names, "NAMI BELLMERE")
		assert.Contains(t, names, "USOPP MONTBLANC")
		assert.Contains(t, names, "TONY TONY CHOPPER")
		assert.Contains(t, names, "ROBIN NICO")
		assert.Contains(t, names, "CUTTY FLAM")
		assert.Contains(t, names, "GARP D MONKEY")
		assert.Contains(t, names, "ACE D PORTGAS")
		assert.Contains(t, names, "SENGOKU BUDDHA")
	}

	t.Log("List the persons as user 1.")
	{
		var output downballotapi.ListPersonsResponse
		err := user1Client.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/person", nil, &output)
		require.NoError(t, err)

		t.Logf("Persons: %v", output.Persons)
		assert.Len(t, output.Persons, 2)
		names := []string{}
		for _, person := range output.Persons {
			names = append(names, person.Fields["name"])
		}
		assert.Contains(t, names, "LUFFY D MONKEY")
		assert.Contains(t, names, "GARP D MONKEY")
	}

	t.Log("List the persons as user 2.")
	{
		var output downballotapi.ListPersonsResponse
		err := user2Client.Do(ctx, http.MethodGet, "/api/v1/organization/"+organizationId+"/person", nil, &output)
		require.NoError(t, err)

		t.Logf("Persons: %v", output.Persons)
		assert.Len(t, output.Persons, 3)
		names := []string{}
		for _, person := range output.Persons {
			names = append(names, person.Fields["name"])
		}
		assert.Contains(t, names, "LUFFY D MONKEY")
		assert.Contains(t, names, "GARP D MONKEY")
		assert.Contains(t, names, "SENGOKU BUDDHA")
	}
}
