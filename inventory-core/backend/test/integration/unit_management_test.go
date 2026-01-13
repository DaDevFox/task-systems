package integrationpackage integrationpackage integration



import (

    "context"

    "os"import (import (

    "testing"

    "context"	"context"

    "github.com/stretchr/testify/assert"

    "github.com/stretchr/testify/require"    "os"	"os"



    "github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"    "testing"	"testing"

    "github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"

    pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"

    "github.com/DaDevFox/task-systems/shared/events"

    "github.com/sirupsen/logrus"    "github.com/stretchr/testify/assert"	"github.com/sirupsen/logrus"

)

    "github.com/stretchr/testify/require"	"github.com/stretchr/testify/assert"

func TestUnitManagementIntegration(t *testing.T) {

    repo, cleanup := setupTestRepository(t)	"github.com/stretchr/testify/require"

    defer cleanup()

    "github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"

    eventBus := events.GetGlobalBus("test-unit-management")

    logger := logrus.New()    "github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"

    logger.SetLevel(logrus.ErrorLevel)

    pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"

    inventoryService := service.NewInventoryService(repo, eventBus, logger)

    inventoryService.DisableAuthForTesting()    "github.com/DaDevFox/task-systems/shared/events"	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"

    ctx := context.Background()

    "github.com/sirupsen/logrus"	"github.com/DaDevFox/task-systems/shared/events"

    t.Run("ListUnits", func(t *testing.T) {

        resp, err := inventoryService.ListUnits(ctx, &pb.ListUnitsRequest{})))

        require.NoError(t, err)

        require.NotNil(t, resp)

        assert.NotEmpty(t, resp.Units)

    })func TestUnitManagementIntegration(t *testing.T) {const (



    t.Run("AddAndRetrieveUnit", func(t *testing.T) {    repo, cleanup := setupTestRepository()	imperialVolumeDescription = "Imperial volume measurement"

        const (

            unitName        = "Tablespoons"    defer cleanup()	testUnitName             = "Test Unit"

            unitSymbol      = "tbsp"

            unitDescription = "Imperial volume measurement")

            unitCategory    = "volume"

        )    eventBus := events.GetGlobalBus("test-unit-management")



        createResp, err := inventoryService.AddUnit(ctx, &pb.AddUnitRequest{    logger := logrus.New()func TestUnitManagementIntegration(t *testing.T) {

            Name:                 unitName,

            Symbol:               unitSymbol,    logger.SetLevel(logrus.ErrorLevel)	repo, cleanup := setupTestRepository()

            Description:          unitDescription,

            BaseConversionFactor: 0.0147868,	defer cleanup()

            Category:             unitCategory,

            Metadata:             map[string]string{"type": "cooking"},    inventoryService := service.NewInventoryService(repo, eventBus, logger)

        })

        require.NoError(t, err)    inventoryService.DisableAuthForTesting()	eventBus := events.GetGlobalBus("test")

        require.NotNil(t, createResp)

        require.NotNil(t, createResp.Unit)    ctx := context.Background()	logger := logrus.New()



        assert.Equal(t, unitName, createResp.Unit.Name)	logger.SetLevel(logrus.ErrorLevel)

        assert.Equal(t, unitSymbol, createResp.Unit.Symbol)

        assert.Equal(t, unitDescription, createResp.Unit.Description)    t.Run("ListUnits", func(t *testing.T) {

        assert.Equal(t, unitCategory, createResp.Unit.Category)

        resp, err := inventoryService.ListUnits(ctx, &pb.ListUnitsRequest{})		})

        fetched, err := inventoryService.GetUnit(ctx, &pb.GetUnitRequest{UnitId: createResp.Unit.Id})

        require.NoError(t, err)        require.NoError(t, err)

        require.NotNil(t, fetched)

        require.NotNil(t, fetched.Unit)        require.NotNil(t, resp)		t.Run("GetUnit with non-existent ID", func(t *testing.T) {

        assert.Equal(t, createResp.Unit.Id, fetched.Unit.Id)

        assert.Equal(t, unitName, fetched.Unit.Name)        assert.NotEmpty(t, resp.Units)			req := &pb.GetUnitRequest{UnitId: "non-existent"}

    })

    })			_, err := inventoryService.GetUnit(ctx, req)

    t.Run("UpdateUnit", func(t *testing.T) {

        createResp, err := inventoryService.AddUnit(ctx, &pb.AddUnitRequest{			assert.Error(t, err)

            Name:                 "Updatable",

            Symbol:               "upd",    t.Run("AddAndRetrieveUnit", func(t *testing.T) {		})

            BaseConversionFactor: 1.0,

            Category:             "test",        const (	})

        })

        require.NoError(t, err)            unitName        = "Tablespoons"}



        updateResp, err := inventoryService.UpdateUnit(ctx, &pb.UpdateUnitRequest{            unitSymbol      = "tbsp"

            UnitId:               createResp.Unit.Id,

            Name:                 "Updated",            unitDescription = "Imperial volume measurement"func setupTestRepository() (repository.InventoryRepository, func()) {

            Symbol:               "upd2",

            Description:          "Updated description",            unitCategory    = "volume"	tempDir, err := os.MkdirTemp("", "badger_test_*")

            BaseConversionFactor: 2.0,

            Category:             "updated",        )	if err != nil {

            Metadata:             map[string]string{"updated": "true"},

        })		panic(err)

        require.NoError(t, err)

        require.NotNil(t, updateResp)        createResp, err := inventoryService.AddUnit(ctx, &pb.AddUnitRequest{	}

        require.NotNil(t, updateResp.Unit)

        assert.True(t, updateResp.UnitChanged)            Name:                 unitName,

        assert.Equal(t, "Updated", updateResp.Unit.Name)

        assert.Equal(t, "updated", updateResp.Unit.Category)            Symbol:               unitSymbol,	repo, err := repository.NewBadgerInventoryRepository(tempDir)

    })

            Description:          unitDescription,	if err != nil {

    t.Run("DeleteUnit", func(t *testing.T) {

        createResp, err := inventoryService.AddUnit(ctx, &pb.AddUnitRequest{            BaseConversionFactor: 0.0147868,		os.RemoveAll(tempDir)

            Name:                 "Deletable",

            Symbol:               "del",            Category:             unitCategory,		panic(err)

            BaseConversionFactor: 1.0,

            Category:             "test",            Metadata:             map[string]string{"type": "cooking"},	}

        })

        require.NoError(t, err)        })



        _, err = inventoryService.DeleteUnit(ctx, &pb.DeleteUnitRequest{        require.NoError(t, err)	cleanup := func() {

            UnitId: createResp.Unit.Id,

            Force:  true,        require.NotNil(t, createResp)		repo.Close()

        })

        require.NoError(t, err)        require.NotNil(t, createResp.Unit)		os.RemoveAll(tempDir)



        _, err = inventoryService.GetUnit(ctx, &pb.GetUnitRequest{UnitId: createResp.Unit.Id})	}

        assert.Error(t, err)

    })        assert.Equal(t, unitName, createResp.Unit.Name)



    t.Run("ValidationFailures", func(t *testing.T) {        assert.Equal(t, unitSymbol, createResp.Unit.Symbol)	return repo, cleanup

        _, err := inventoryService.AddUnit(ctx, &pb.AddUnitRequest{

            Name:                 "",        assert.Equal(t, unitDescription, createResp.Unit.Description)}

            Symbol:               "bad",

            BaseConversionFactor: 1.0,        assert.Equal(t, unitCategory, createResp.Unit.Category)

        })

        assert.Error(t, err)	inventoryService.DisableAuthForTesting()



        _, err = inventoryService.UpdateUnit(ctx, &pb.UpdateUnitRequest{})        fetched, err := inventoryService.GetUnit(ctx, &pb.GetUnitRequest{UnitId: createResp.Unit.Id})

        assert.Error(t, err)

    })        require.NoError(t, err)	ctx := context.Background()

}

        require.NotNil(t, fetched)

func setupTestRepository(t *testing.T) (repository.InventoryRepository, func()) {

    t.Helper()        require.NotNil(t, fetched.Unit)



    tempDir, err := os.MkdirTemp("", "unit-management-test-*")        assert.Equal(t, createResp.Unit.Id, fetched.Unit.Id)

    require.NoError(t, err)

        assert.Equal(t, unitName, fetched.Unit.Name)	t.Run("ListUnits", func(t *testing.T) {const (const (

    repo, err := repository.NewBadgerInventoryRepository(tempDir)

    require.NoError(t, err)    })

package integration

    cleanup := func() {

        repo.Close()import (

        os.RemoveAll(tempDir)	"context"

    }	"os"

	"testing"

    return repo, cleanup

}	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
	"github.com/sirupsen/logrus"
)

func TestUnitManagementIntegration(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	defer cleanup()

	eventBus := events.GetGlobalBus("test-unit-management")
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	inventoryService := service.NewInventoryService(repo, eventBus, logger)
	inventoryService.DisableAuthForTesting()
	ctx := context.Background()

	t.Run("ListUnits", func(t *testing.T) {
		resp, err := inventoryService.ListUnits(ctx, &pb.ListUnitsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.Units)
	})

	t.Run("AddAndRetrieveUnit", func(t *testing.T) {
		const (
			unitName        = "Tablespoons"
			unitSymbol      = "tbsp"
			unitDescription = "Imperial volume measurement"
			unitCategory    = "volume"
		)

		createResp, err := inventoryService.AddUnit(ctx, &pb.AddUnitRequest{
			Name:                 unitName,
			Symbol:               unitSymbol,
			Description:          unitDescription,
			BaseConversionFactor: 0.0147868,
			Category:             unitCategory,
			Metadata:             map[string]string{"type": "cooking"},
		})
		require.NoError(t, err)
		require.NotNil(t, createResp)
		require.NotNil(t, createResp.Unit)

		assert.Equal(t, unitName, createResp.Unit.Name)
		assert.Equal(t, unitSymbol, createResp.Unit.Symbol)
		assert.Equal(t, unitDescription, createResp.Unit.Description)
		assert.Equal(t, unitCategory, createResp.Unit.Category)

		fetched, err := inventoryService.GetUnit(ctx, &pb.GetUnitRequest{UnitId: createResp.Unit.Id})
		require.NoError(t, err)
		require.NotNil(t, fetched)
		require.NotNil(t, fetched.Unit)
		assert.Equal(t, createResp.Unit.Id, fetched.Unit.Id)
		assert.Equal(t, unitName, fetched.Unit.Name)
	})

	t.Run("UpdateUnit", func(t *testing.T) {
		createResp, err := inventoryService.AddUnit(ctx, &pb.AddUnitRequest{
			Name:                 "Updatable",
			Symbol:               "upd",
			BaseConversionFactor: 1.0,
			Category:             "test",
		})
		require.NoError(t, err)

		updateResp, err := inventoryService.UpdateUnit(ctx, &pb.UpdateUnitRequest{
			UnitId:               createResp.Unit.Id,
			Name:                 "Updated",
			Symbol:               "upd2",
			Description:          "Updated description",
			BaseConversionFactor: 2.0,
			Category:             "updated",
			Metadata:             map[string]string{"updated": "true"},
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)
		require.NotNil(t, updateResp.Unit)
		assert.True(t, updateResp.UnitChanged)
		assert.Equal(t, "Updated", updateResp.Unit.Name)
		assert.Equal(t, "updated", updateResp.Unit.Category)
	})

	t.Run("DeleteUnit", func(t *testing.T) {
		createResp, err := inventoryService.AddUnit(ctx, &pb.AddUnitRequest{
			Name:                 "Deletable",
			Symbol:               "del",
			BaseConversionFactor: 1.0,
			Category:             "test",
		})
		require.NoError(t, err)

		_, err = inventoryService.DeleteUnit(ctx, &pb.DeleteUnitRequest{
			UnitId: createResp.Unit.Id,
			Force:  true,
		})
		require.NoError(t, err)

		_, err = inventoryService.GetUnit(ctx, &pb.GetUnitRequest{UnitId: createResp.Unit.Id})
		assert.Error(t, err)
	})

	t.Run("ValidationFailures", func(t *testing.T) {
		_, err := inventoryService.AddUnit(ctx, &pb.AddUnitRequest{
			Name:                 "",
			Symbol:               "bad",
			BaseConversionFactor: 1.0,
		})
		assert.Error(t, err)

		_, err = inventoryService.UpdateUnit(ctx, &pb.UpdateUnitRequest{})
		assert.Error(t, err)
	})
}

func setupTestRepository(t *testing.T) (repository.InventoryRepository, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "unit-management-test-*")
	require.NoError(t, err)

	repo, err := repository.NewBadgerInventoryRepository(tempDir)
	require.NoError(t, err)

	cleanup := func() {
		repo.Close()
		os.RemoveAll(tempDir)
	}

	return repo, cleanup
}
}

		assert.Equal(t, 0.0147868, resp.Unit.BaseConversionFactor)

		assert.Equal(t, "volume", resp.Unit.Category)        resp, err := inventoryService.ListUnits(ctx, req)		"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"

		assert.Equal(t, "cooking", resp.Unit.Metadata["type"])

	})		pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"



	t.Run("GetUnit", func(t *testing.T) {        require.NoError(t, err)		"github.com/DaDevFox/task-systems/shared/events"

		addReq := &pb.AddUnitRequest{

			Name:                 "Teaspoons",        require.NotNil(t, resp)	)

			Symbol:               "tsp",

			Description:          imperialVolumeDescription,        assert.Greater(t, len(resp.Units), 0, "Should have default units")

			BaseConversionFactor: 0.00492892,

			Category:             "volume",	const (

		}

        unitIDs := make(map[string]bool)		imperialVolumeDescription = "Imperial volume measurement"

		addResp, err := inventoryService.AddUnit(ctx, addReq)

		require.NoError(t, err)        for _, unit := range resp.Units {		testUnitName             = "Test Unit"

		unitID := addResp.Unit.Id

            unitIDs[unit.Id] = true	)

		getReq := &pb.GetUnitRequest{UnitId: unitID}

		getResp, err := inventoryService.GetUnit(ctx, getReq)        }



		require.NoError(t, err)	func TestUnitManagementIntegration(t *testing.T) {

		require.NotNil(t, getResp)

		require.NotNil(t, getResp.Unit)        assert.True(t, unitIDs["kg"], "Should have kg unit")		repo, cleanup := setupTestRepository()



		assert.Equal(t, unitID, getResp.Unit.Id)        assert.True(t, unitIDs["g"], "Should have g unit")		defer cleanup()

		assert.Equal(t, "Teaspoons", getResp.Unit.Name)

		assert.Equal(t, "tsp", getResp.Unit.Symbol)        assert.True(t, unitIDs["l"], "Should have l unit")

	})

    })		eventBus := events.GetGlobalBus("test")

	t.Run("UpdateUnit", func(t *testing.T) {

		addReq := &pb.AddUnitRequest{		logger := logrus.New()

			Name:                 testUnitName,

			Symbol:               "test",    t.Run("AddUnit", func(t *testing.T) {		logger.SetLevel(logrus.ErrorLevel)

			BaseConversionFactor: 1,

			Category:             "test",        req := &pb.AddUnitRequest{

		}

            Name:                 "Tablespoons",		inventoryService := service.NewInventoryService(repo, eventBus, logger)

		addResp, err := inventoryService.AddUnit(ctx, addReq)

		require.NoError(t, err)            Symbol:               "tbsp",		inventoryService.DisableAuthForTesting()

		unitID := addResp.Unit.Id

            Description:          imperialVolumeDescription,		ctx := context.Background()

		updateReq := &pb.UpdateUnitRequest{

			UnitId:               unitID,            BaseConversionFactor: 0.0147868,

			Name:                 "Updated Test Unit",

			Symbol:               "utest",            Category:             "volume",		t.Run("ListUnits", func(t *testing.T) {

			Description:          "Updated description",

			BaseConversionFactor: 2,            Metadata:             map[string]string{"type": "cooking"},			req := &pb.ListUnitsRequest{}

			Category:             "updated",

			Metadata:             map[string]string{"updated": "true"},        }			resp, err := inventoryService.ListUnits(ctx, req)

		}



		updateResp, err := inventoryService.UpdateUnit(ctx, updateReq)

        resp, err := inventoryService.AddUnit(ctx, req)			require.NoError(t, err)

		require.NoError(t, err)

		require.NotNil(t, updateResp)			require.NotNil(t, resp)

		require.NotNil(t, updateResp.Unit)

		assert.True(t, updateResp.UnitChanged)        require.NoError(t, err)			assert.Greater(t, len(resp.Units), 0, "Should have default units")



		assert.Equal(t, unitID, updateResp.Unit.Id)        require.NotNil(t, resp)

		assert.Equal(t, "Updated Test Unit", updateResp.Unit.Name)

		assert.Equal(t, "utest", updateResp.Unit.Symbol)        require.NotNil(t, resp.Unit)			unitIDs := make(map[string]bool)

	})

			for _, unit := range resp.Units {

	t.Run("DeleteUnit", func(t *testing.T) {

		addReq := &pb.AddUnitRequest{        assert.Equal(t, "Tablespoons", resp.Unit.Name)				unitIDs[unit.Id] = true

			Name:                 "Deletable Unit",

			Symbol:               "del",        assert.Equal(t, "tbsp", resp.Unit.Symbol)			}

			BaseConversionFactor: 1,

			Category:             "test",        assert.Equal(t, imperialVolumeDescription, resp.Unit.Description)

		}

        assert.Equal(t, 0.0147868, resp.Unit.BaseConversionFactor)			assert.True(t, unitIDs["kg"], "Should have kg unit")

		addResp, err := inventoryService.AddUnit(ctx, addReq)

		require.NoError(t, err)        assert.Equal(t, "volume", resp.Unit.Category)			assert.True(t, unitIDs["g"], "Should have g unit")

		unitID := addResp.Unit.Id

        assert.Equal(t, "cooking", resp.Unit.Metadata["type"])			assert.True(t, unitIDs["l"], "Should have l unit")

		deleteReq := &pb.DeleteUnitRequest{

			UnitId: unitID,    })		})

			Force:  true,

		}



		deleteResp, err := inventoryService.DeleteUnit(ctx, deleteReq)    t.Run("GetUnit", func(t *testing.T) {		t.Run("AddUnit", func(t *testing.T) {



		require.NoError(t, err)        addReq := &pb.AddUnitRequest{			req := &pb.AddUnitRequest{

		require.NotNil(t, deleteResp)

		assert.True(t, deleteResp.UnitDeleted)            Name:                 "Teaspoons",				Name:                 "Tablespoons",

		assert.Equal(t, unitID, deleteResp.DeletedUnitId)

            Symbol:               "tsp",				Symbol:               "tbsp",

		getReq := &pb.GetUnitRequest{UnitId: unitID}

		_, err = inventoryService.GetUnit(ctx, getReq)            Description:          imperialVolumeDescription,				Description:          imperialVolumeDescription,

		assert.Error(t, err)

	})            BaseConversionFactor: 0.00492892,				BaseConversionFactor: 0.0147868,



	t.Run("ValidationErrors", func(t *testing.T) {            Category:             "volume",				Category:             "volume",

		t.Run("AddUnit with empty name", func(t *testing.T) {

			req := &pb.AddUnitRequest{        }				Metadata:             map[string]string{"type": "cooking"},

				Name:                 "",

				Symbol:               "test",			}

				BaseConversionFactor: 1,

			}        addResp, err := inventoryService.AddUnit(ctx, addReq)



			_, err := inventoryService.AddUnit(ctx, req)        require.NoError(t, err)			resp, err := inventoryService.AddUnit(ctx, req)

			assert.Error(t, err)

		})        unitID := addResp.Unit.Id



		t.Run("AddUnit with empty symbol", func(t *testing.T) {			require.NoError(t, err)

			req := &pb.AddUnitRequest{

				Name:                 testUnitName,        getReq := &pb.GetUnitRequest{UnitId: unitID}			require.NotNil(t, resp)

				Symbol:               "",

				BaseConversionFactor: 1,        getResp, err := inventoryService.GetUnit(ctx, getReq)			require.NotNil(t, resp.Unit)

			}



			_, err := inventoryService.AddUnit(ctx, req)

			assert.Error(t, err)        require.NoError(t, err)			assert.Equal(t, "Tablespoons", resp.Unit.Name)

		})

        require.NotNil(t, getResp)			assert.Equal(t, "tbsp", resp.Unit.Symbol)

		t.Run("AddUnit with zero conversion factor", func(t *testing.T) {

			req := &pb.AddUnitRequest{        require.NotNil(t, getResp.Unit)			assert.Equal(t, imperialVolumeDescription, resp.Unit.Description)

				Name:                 testUnitName,

				Symbol:               "test",			assert.Equal(t, 0.0147868, resp.Unit.BaseConversionFactor)

				BaseConversionFactor: 0,

			}        assert.Equal(t, unitID, getResp.Unit.Id)			assert.Equal(t, "volume", resp.Unit.Category)



			_, err := inventoryService.AddUnit(ctx, req)        assert.Equal(t, "Teaspoons", getResp.Unit.Name)			assert.Equal(t, "cooking", resp.Unit.Metadata["type"])

			assert.Error(t, err)

		})        assert.Equal(t, "tsp", getResp.Unit.Symbol)		})



		t.Run("GetUnit with empty ID", func(t *testing.T) {    })

			req := &pb.GetUnitRequest{UnitId: ""}

			_, err := inventoryService.GetUnit(ctx, req)		t.Run("GetUnit", func(t *testing.T) {

			assert.Error(t, err)

		})    t.Run("UpdateUnit", func(t *testing.T) {			addReq := &pb.AddUnitRequest{



		t.Run("GetUnit with non-existent ID", func(t *testing.T) {        addReq := &pb.AddUnitRequest{				Name:                 "Teaspoons",

			req := &pb.GetUnitRequest{UnitId: "non-existent"}

			_, err := inventoryService.GetUnit(ctx, req)            Name:                 testUnitName,				Symbol:               "tsp",

			assert.Error(t, err)

		})            Symbol:               "test",				Description:          imperialVolumeDescription,

	})

}            BaseConversionFactor: 1,				BaseConversionFactor: 0.00492892,



func setupTestRepository() (repository.InventoryRepository, func()) {            Category:             "test",				Category:             "volume",

	tempDir, err := os.MkdirTemp("", "badger_test_*")

	if err != nil {        }			}

		panic(err)

	}



	repo, err := repository.NewBadgerInventoryRepository(tempDir)        addResp, err := inventoryService.AddUnit(ctx, addReq)			addResp, err := inventoryService.AddUnit(ctx, addReq)

	if err != nil {

		os.RemoveAll(tempDir)        require.NoError(t, err)			require.NoError(t, err)

		panic(err)

	}        unitID := addResp.Unit.Id			unitID := addResp.Unit.Id



	cleanup := func() {

		repo.Close()

		os.RemoveAll(tempDir)        updateReq := &pb.UpdateUnitRequest{			getReq := &pb.GetUnitRequest{UnitId: unitID}

	}

            UnitId:               unitID,			getResp, err := inventoryService.GetUnit(ctx, getReq)

	return repo, cleanup

}            Name:                 "Updated Test Unit",


            Symbol:               "utest",			require.NoError(t, err)

            Description:          "Updated description",			require.NotNil(t, getResp)

            BaseConversionFactor: 2,			require.NotNil(t, getResp.Unit)

            Category:             "updated",

            Metadata:             map[string]string{"updated": "true"},			assert.Equal(t, unitID, getResp.Unit.Id)

        }			assert.Equal(t, "Teaspoons", getResp.Unit.Name)

			assert.Equal(t, "tsp", getResp.Unit.Symbol)

        updateResp, err := inventoryService.UpdateUnit(ctx, updateReq)		})



        require.NoError(t, err)		t.Run("UpdateUnit", func(t *testing.T) {

        require.NotNil(t, updateResp)			addReq := &pb.AddUnitRequest{

        require.NotNil(t, updateResp.Unit)				Name:                 testUnitName,

        assert.True(t, updateResp.UnitChanged)				Symbol:               "test",

				BaseConversionFactor: 1,

        assert.Equal(t, unitID, updateResp.Unit.Id)				Category:             "test",

        assert.Equal(t, "Updated Test Unit", updateResp.Unit.Name)			}

        assert.Equal(t, "utest", updateResp.Unit.Symbol)

    })			addResp, err := inventoryService.AddUnit(ctx, addReq)

			require.NoError(t, err)

    t.Run("DeleteUnit", func(t *testing.T) {			unitID := addResp.Unit.Id

        addReq := &pb.AddUnitRequest{

            Name:                 "Deletable Unit",			updateReq := &pb.UpdateUnitRequest{

            Symbol:               "del",				UnitId:               unitID,

            BaseConversionFactor: 1,				Name:                 "Updated Test Unit",

            Category:             "test",				Symbol:               "utest",

        }				Description:          "Updated description",

				BaseConversionFactor: 2,

        addResp, err := inventoryService.AddUnit(ctx, addReq)				Category:             "updated",

        require.NoError(t, err)				Metadata:             map[string]string{"updated": "true"},

        unitID := addResp.Unit.Id			}



        deleteReq := &pb.DeleteUnitRequest{			updateResp, err := inventoryService.UpdateUnit(ctx, updateReq)

            UnitId: unitID,

            Force:  true,			require.NoError(t, err)

        }			require.NotNil(t, updateResp)

			require.NotNil(t, updateResp.Unit)

        deleteResp, err := inventoryService.DeleteUnit(ctx, deleteReq)			assert.True(t, updateResp.UnitChanged)



        require.NoError(t, err)			assert.Equal(t, unitID, updateResp.Unit.Id)

        require.NotNil(t, deleteResp)			assert.Equal(t, "Updated Test Unit", updateResp.Unit.Name)

        assert.True(t, deleteResp.UnitDeleted)			assert.Equal(t, "utest", updateResp.Unit.Symbol)

        assert.Equal(t, unitID, deleteResp.DeletedUnitId)		})



        getReq := &pb.GetUnitRequest{UnitId: unitID}		t.Run("DeleteUnit", func(t *testing.T) {

        _, err = inventoryService.GetUnit(ctx, getReq)			addReq := &pb.AddUnitRequest{

        assert.Error(t, err)				Name:                 "Deletable Unit",

    })				Symbol:               "del",

				BaseConversionFactor: 1,

    t.Run("ValidationErrors", func(t *testing.T) {				Category:             "test",

        t.Run("AddUnit with empty name", func(t *testing.T) {			}

            req := &pb.AddUnitRequest{

                Name:                 "",			addResp, err := inventoryService.AddUnit(ctx, addReq)

                Symbol:               "test",			require.NoError(t, err)

                BaseConversionFactor: 1,			unitID := addResp.Unit.Id

            }

			deleteReq := &pb.DeleteUnitRequest{

            _, err := inventoryService.AddUnit(ctx, req)				UnitId: unitID,

            assert.Error(t, err)				Force:  true,

        })			}



        t.Run("AddUnit with empty symbol", func(t *testing.T) {			deleteResp, err := inventoryService.DeleteUnit(ctx, deleteReq)

            req := &pb.AddUnitRequest{

                Name:                 testUnitName,			require.NoError(t, err)

                Symbol:               "",			require.NotNil(t, deleteResp)

                BaseConversionFactor: 1,			assert.True(t, deleteResp.UnitDeleted)

            }			assert.Equal(t, unitID, deleteResp.DeletedUnitId)



            _, err := inventoryService.AddUnit(ctx, req)			getReq := &pb.GetUnitRequest{UnitId: unitID}

            assert.Error(t, err)			_, err = inventoryService.GetUnit(ctx, getReq)

        })			assert.Error(t, err)

		})

        t.Run("AddUnit with zero conversion factor", func(t *testing.T) {

            req := &pb.AddUnitRequest{		t.Run("ValidationErrors", func(t *testing.T) {

                Name:                 testUnitName,			t.Run("AddUnit with empty name", func(t *testing.T) {

                Symbol:               "test",				req := &pb.AddUnitRequest{

                BaseConversionFactor: 0,					Name:                 "",

            }					Symbol:               "test",

					BaseConversionFactor: 1,

            _, err := inventoryService.AddUnit(ctx, req)				}

            assert.Error(t, err)

        })				_, err := inventoryService.AddUnit(ctx, req)

				assert.Error(t, err)

        t.Run("GetUnit with empty ID", func(t *testing.T) {			})

            req := &pb.GetUnitRequest{UnitId: ""}

            _, err := inventoryService.GetUnit(ctx, req)			t.Run("AddUnit with empty symbol", func(t *testing.T) {

            assert.Error(t, err)				req := &pb.AddUnitRequest{

        })					Name:                 testUnitName,

					Symbol:               "",

        t.Run("GetUnit with non-existent ID", func(t *testing.T) {					BaseConversionFactor: 1,

            req := &pb.GetUnitRequest{UnitId: "non-existent"}				}

            _, err := inventoryService.GetUnit(ctx, req)

            assert.Error(t, err)				_, err := inventoryService.AddUnit(ctx, req)

        })				assert.Error(t, err)

    })			})

}

			t.Run("AddUnit with zero conversion factor", func(t *testing.T) {

func setupTestRepository() (repository.InventoryRepository, func()) {				req := &pb.AddUnitRequest{

    tempDir, err := os.MkdirTemp("", "badger_test_*")					Name:                 testUnitName,

    if err != nil {					Symbol:               "test",

        panic(err)					BaseConversionFactor: 0,

    }				}



    repo, err := repository.NewBadgerInventoryRepository(tempDir)				_, err := inventoryService.AddUnit(ctx, req)

    if err != nil {				assert.Error(t, err)

        os.RemoveAll(tempDir)			})

        panic(err)

    }			t.Run("GetUnit with empty ID", func(t *testing.T) {

				req := &pb.GetUnitRequest{UnitId: ""}

    cleanup := func() {				_, err := inventoryService.GetUnit(ctx, req)

        repo.Close()				assert.Error(t, err)

        os.RemoveAll(tempDir)			})

    }

			t.Run("GetUnit with non-existent ID", func(t *testing.T) {

    return repo, cleanup				req := &pb.GetUnitRequest{UnitId: "non-existent"}

}				_, err := inventoryService.GetUnit(ctx, req)

				assert.Error(t, err)
			})
		})
	}

	func setupTestRepository() (repository.InventoryRepository, func()) {
		tempDir, err := os.MkdirTemp("", "badger_test_*")
		if err != nil {
			panic(err)
		}

		repo, err := repository.NewBadgerInventoryRepository(tempDir)
		if err != nil {
			os.RemoveAll(tempDir)
			panic(err)
		}

		cleanup := func() {
			repo.Close()
			os.RemoveAll(tempDir)
		}

		return repo, cleanup
	}
	return repo, cleanup
