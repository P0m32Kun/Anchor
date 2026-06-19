package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func setupTestAsset(t *testing.T, q *Queries) (*models.Asset, *models.Project) {
	t.Helper()
	now := time.Now().UTC()
	project := &models.Project{
		ID: "proj-asset", Name: "asset-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateProject(project); err != nil {
		t.Fatalf("create project: %v", err)
	}
	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: project.ID,
		Type: models.AssetTypeDomain, Value: "foo.example.com",
		NormalizedValue: "foo.example.com",
		SourceTools:     []string{"subfinder"},
		FirstSeen: now, LastSeen: now,
	}
	if err := q.CreateAsset(asset); err != nil {
		t.Fatalf("create asset: %v", err)
	}
	return asset, project
}

func TestUpdateAssetLastSeen(t *testing.T) {
	q := New(openTestDB(t))
	asset, _ := setupTestAsset(t, q)

	newTime := time.Now().UTC().Add(time.Hour)
	if err := q.UpdateAssetLastSeen(asset.ID, newTime, []string{"httpx", "nuclei"}); err != nil {
		t.Fatalf("UpdateAssetLastSeen: %v", err)
	}

	got, err := q.GetAssetByID(asset.ID)
	if err != nil {
		t.Fatalf("GetAssetByID: %v", err)
	}
	if got == nil {
		t.Fatal("asset not found")
	}
	if len(got.SourceTools) != 2 {
		t.Errorf("source_tools len = %d, want 2", len(got.SourceTools))
	}
}

func TestListAssetsByProjectPaginated(t *testing.T) {
	q := New(openTestDB(t))
	_, project := setupTestAsset(t, q)
	now := time.Now().UTC()

	for i := 0; i < 5; i++ {
		if err := q.CreateAsset(&models.Asset{
			ID: util.GenerateID(), ProjectID: project.ID,
			Type: models.AssetTypeDomain, Value: "d"+string(rune('a'+i))+".example.com",
			NormalizedValue: "d" + string(rune('a'+i)) + ".example.com",
			FirstSeen: now, LastSeen: now,
		}); err != nil {
			t.Fatalf("create asset %d: %v", i, err)
		}
	}

	// Total should be 6 (1 from setupTestAsset + 5)
	all, err := q.ListAssetsByProjectPaginated(project.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListAssetsByProjectPaginated: %v", err)
	}
	if len(all) != 6 {
		t.Errorf("total = %d, want 6", len(all))
	}

	// Paginated: limit 2, offset 1
	page, err := q.ListAssetsByProjectPaginated(project.ID, 2, 1)
	if err != nil {
		t.Fatalf("ListAssetsByProjectPaginated page: %v", err)
	}
	if len(page) != 2 {
		t.Errorf("page len = %d, want 2", len(page))
	}
}

func TestCreatePort_AndListByAsset(t *testing.T) {
	q := New(openTestDB(t))
	asset, _ := setupTestAsset(t, q)
	now := time.Now().UTC()

	port := &models.Port{
		ID: util.GenerateID(), AssetID: asset.ID, Port: 443,
		Protocol: "tcp", State: "open", SourceTool: "naabu", CreatedAt: now,
	}
	if err := q.CreatePort(port); err != nil {
		t.Fatalf("CreatePort: %v", err)
	}

	ports, err := q.ListPortsByAsset(asset.ID)
	if err != nil {
		t.Fatalf("ListPortsByAsset: %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("ports len = %d, want 1", len(ports))
	}
	if ports[0].Port != 443 {
		t.Errorf("port = %d, want 443", ports[0].Port)
	}
}

func TestListPortsByProject(t *testing.T) {
	q := New(openTestDB(t))
	asset, project := setupTestAsset(t, q)
	now := time.Now().UTC()

	if err := q.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: asset.ID, Port: 80,
		Protocol: "tcp", State: "open", SourceTool: "naabu", CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePort: %v", err)
	}
	if err := q.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: asset.ID, Port: 443,
		Protocol: "tcp", State: "open", SourceTool: "naabu", CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePort: %v", err)
	}

	ports, err := q.ListPortsByProject(project.ID)
	if err != nil {
		t.Fatalf("ListPortsByProject: %v", err)
	}
	if len(ports) != 2 {
		t.Errorf("ports len = %d, want 2", len(ports))
	}
}

func TestCountPortsByProject(t *testing.T) {
	q := New(openTestDB(t))
	asset, project := setupTestAsset(t, q)
	now := time.Now().UTC()

	if err := q.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: asset.ID, Port: 80,
		Protocol: "tcp", State: "open", SourceTool: "naabu", CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePort: %v", err)
	}

	count, err := q.CountPortsByProject(project.ID)
	if err != nil {
		t.Fatalf("CountPortsByProject: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestPortExists(t *testing.T) {
	q := New(openTestDB(t))
	asset, _ := setupTestAsset(t, q)
	now := time.Now().UTC()

	if err := q.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: asset.ID, Port: 8080,
		Protocol: "tcp", State: "open", SourceTool: "naabu", CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePort: %v", err)
	}

	exists, err := q.PortExists(asset.ID, 8080)
	if err != nil {
		t.Fatalf("PortExists: %v", err)
	}
	if !exists {
		t.Error("expected port 8080 to exist")
	}

	exists, err = q.PortExists(asset.ID, 9999)
	if err != nil {
		t.Fatalf("PortExists 9999: %v", err)
	}
	if exists {
		t.Error("expected port 9999 not to exist")
	}
}

func TestCreateService_AndListByAsset(t *testing.T) {
	q := New(openTestDB(t))
	asset, _ := setupTestAsset(t, q)
	now := time.Now().UTC()

	svc := &models.Service{
		ID: util.GenerateID(), AssetID: asset.ID, Name: "https",
		Product: "nginx", Version: "1.21", Confidence: 95,
		SourceTool: "nmap", CreatedAt: now,
	}
	if err := q.CreateService(svc); err != nil {
		t.Fatalf("CreateService: %v", err)
	}

	svcs, err := q.ListServicesByAsset(asset.ID)
	if err != nil {
		t.Fatalf("ListServicesByAsset: %v", err)
	}
	if len(svcs) != 1 {
		t.Fatalf("services len = %d, want 1", len(svcs))
	}
	if svcs[0].Product != "nginx" {
		t.Errorf("product = %q, want nginx", svcs[0].Product)
	}
}

func TestCountServicesByProject(t *testing.T) {
	q := New(openTestDB(t))
	asset, project := setupTestAsset(t, q)
	now := time.Now().UTC()

	if err := q.CreateService(&models.Service{
		ID: util.GenerateID(), AssetID: asset.ID, Name: "http",
		SourceTool: "nmap", CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreateService: %v", err)
	}

	count, err := q.CountServicesByProject(project.ID)
	if err != nil {
		t.Fatalf("CountServicesByProject: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestWebEndpoint_CRUD(t *testing.T) {
	q := New(openTestDB(t))
	asset, project := setupTestAsset(t, q)
	now := time.Now().UTC()

	port := 443
	statusCode := 200
	we := &models.WebEndpoint{
		ID: util.GenerateID(), ProjectID: project.ID, AssetID: asset.ID,
		URL: "https://foo.example.com/", Scheme: "https", Host: "foo.example.com",
		Port: &port, Path: "/", StatusCode: &statusCode, Title: "Foo",
		Technologies: []string{"nginx"}, SourceTool: "httpx", CreatedAt: now,
	}
	if err := q.CreateWebEndpoint(we); err != nil {
		t.Fatalf("CreateWebEndpoint: %v", err)
	}

	// ListWebEndpointsByAsset
	endpoints, err := q.ListWebEndpointsByAsset(asset.ID)
	if err != nil {
		t.Fatalf("ListWebEndpointsByAsset: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("endpoints len = %d, want 1", len(endpoints))
	}
	if endpoints[0].Title != "Foo" {
		t.Errorf("title = %q, want Foo", endpoints[0].Title)
	}

	// CountWebEndpointsByProject
	count, err := q.CountWebEndpointsByProject(project.ID)
	if err != nil {
		t.Fatalf("CountWebEndpointsByProject: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// GetWebEndpointByURL
	got, err := q.GetWebEndpointByURL(project.ID, "https://foo.example.com/")
	if err != nil {
		t.Fatalf("GetWebEndpointByURL: %v", err)
	}
	if got == nil {
		t.Fatal("GetWebEndpointByURL returned nil")
	}
	if got.Title != "Foo" {
		t.Errorf("title = %q, want Foo", got.Title)
	}

	// GetWebEndpointByURL not found
	got2, err := q.GetWebEndpointByURL(project.ID, "https://nonexistent.example.com/")
	if err != nil {
		t.Fatalf("GetWebEndpointByURL nonexistent: %v", err)
	}
	if got2 != nil {
		t.Error("expected nil for nonexistent URL")
	}
}

func TestListWebEndpointsByProjectPaginated(t *testing.T) {
	q := New(openTestDB(t))
	asset, project := setupTestAsset(t, q)
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		if err := q.CreateWebEndpoint(&models.WebEndpoint{
			ID: util.GenerateID(), ProjectID: project.ID, AssetID: asset.ID,
			URL: "https://foo.example.com/page" + string(rune('a'+i)),
			Scheme: "https", Host: "foo.example.com",
			Path: "/page" + string(rune('a'+i)), SourceTool: "httpx", CreatedAt: now,
		}); err != nil {
			t.Fatalf("CreateWebEndpoint %d: %v", i, err)
		}
	}

	page, err := q.ListWebEndpointsByProjectPaginated(project.ID, 2, 0)
	if err != nil {
		t.Fatalf("ListWebEndpointsByProjectPaginated: %v", err)
	}
	if len(page) != 2 {
		t.Errorf("page len = %d, want 2", len(page))
	}
}


func TestWebEndpointExistsV2(t *testing.T) {
	q := New(openTestDB(t))
	asset, project := setupTestAsset(t, q)
	now := time.Now().UTC()

	if err := q.CreateWebEndpoint(&models.WebEndpoint{
		ID: util.GenerateID(), ProjectID: project.ID, AssetID: asset.ID,
		URL: "https://foo.example.com/exist", Scheme: "https", Host: "foo.example.com",
		Path: "/exist", SourceTool: "httpx", CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreateWebEndpoint: %v", err)
	}

	exists, err := q.WebEndpointExists(project.ID, "https://foo.example.com/exist")
	if err != nil {
		t.Fatalf("WebEndpointExists: %v", err)
	}
	if !exists {
		t.Error("expected endpoint to exist")
	}

	exists, err = q.WebEndpointExists(project.ID, "https://foo.example.com/nope")
	if err != nil {
		t.Fatalf("WebEndpointExists nope: %v", err)
	}
	if exists {
		t.Error("expected endpoint not to exist")
	}
}

func TestWebEndpoint_Upsert(t *testing.T) {
	q := New(openTestDB(t))
	asset, project := setupTestAsset(t, q)
	now := time.Now().UTC()

	url := "https://foo.example.com/upsert"
	if err := q.CreateWebEndpoint(&models.WebEndpoint{
		ID: "we-1", ProjectID: project.ID, AssetID: asset.ID,
		URL: url, Scheme: "https", Host: "foo.example.com",
		Path: "/upsert", Title: "v1", SourceTool: "httpx", CreatedAt: now,
	}); err != nil {
		t.Fatalf("first CreateWebEndpoint: %v", err)
	}

	// Upsert with different title
	if err := q.CreateWebEndpoint(&models.WebEndpoint{
		ID: "we-2", ProjectID: project.ID, AssetID: asset.ID,
		URL: url, Scheme: "https", Host: "foo.example.com",
		Path: "/upsert", Title: "v2", SourceTool: "httpx", CreatedAt: now,
	}); err != nil {
		t.Fatalf("second CreateWebEndpoint: %v", err)
	}

	got, err := q.GetWebEndpointByURL(project.ID, url)
	if err != nil {
		t.Fatalf("GetWebEndpointByURL: %v", err)
	}
	if got.Title != "v2" {
		t.Errorf("title = %q, want v2 (upsert should update)", got.Title)
	}
	if got.ID != "we-1" {
		t.Errorf("id = %q, want we-1 (upsert should keep original id)", got.ID)
	}
}

func TestListAssetsByProjectV2(t *testing.T) {
	q := New(openTestDB(t))
	_, project := setupTestAsset(t, q)
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		if err := q.CreateAsset(&models.Asset{
			ID: util.GenerateID(), ProjectID: project.ID,
			Type: models.AssetTypeDomain, Value: "list"+string(rune('a'+i))+".example.com",
			NormalizedValue: "list" + string(rune('a'+i)) + ".example.com",
			FirstSeen: now, LastSeen: now,
		}); err != nil {
			t.Fatalf("create asset %d: %v", i, err)
		}
	}

	assets, err := q.ListAssetsByProject(project.ID)
	if err != nil {
		t.Fatalf("ListAssetsByProject: %v", err)
	}
	if len(assets) != 4 { // 1 from setup + 3
		t.Errorf("assets len = %d, want 4", len(assets))
	}
}

func TestCountAssetsByProjectV2(t *testing.T) {
	q := New(openTestDB(t))
	_, project := setupTestAsset(t, q)

	count, err := q.CountAssetsByProject(project.ID)
	if err != nil {
		t.Fatalf("CountAssetsByProject: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestGetAssetByNormalizedValue(t *testing.T) {
	q := New(openTestDB(t))
	_, project := setupTestAsset(t, q)

	got, err := q.GetAssetByNormalizedValue(project.ID, "foo.example.com")
	if err != nil {
		t.Fatalf("GetAssetByNormalizedValue: %v", err)
	}
	if got == nil {
		t.Fatal("asset not found")
	}
	if got.Value != "foo.example.com" {
		t.Errorf("value = %q, want foo.example.com", got.Value)
	}
}
