package records

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/g3/models"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/transport"
)

type Client interface {
	Read(url.Values) (models.G3PaginatedResponse, error)
	Create(models.G3Record) (models.G3MutationResponse, error)
	Update(models.G3Record) (models.G3MutationResponse, error)
	Delete(id string) error
}

type recordsClient struct {
	transport transport.Transport
	path      string // API sub-path for recordsClient
}

func NewRecordsClient(transport transport.Transport) Client {
	return &recordsClient{
		transport: transport,
		path:      "/dns/record",
	}
}

func (r *recordsClient) Read(params url.Values) (models.G3PaginatedResponse, error) {
	page := models.G3PaginatedResponse{}

	err := r.transport.GetJSON(context.Background(), r.path+"s?"+params.Encode(), &page)
	if err != nil {
		return models.G3PaginatedResponse{}, fmt.Errorf("failed to fetch records: %w", err)
	}
	return page, nil
}

func (r *recordsClient) Create(record models.G3Record) (models.G3MutationResponse, error) {
	created := models.G3MutationResponse{}

	err := r.transport.PostJSON(context.Background(), r.path, record, &created)
	if err != nil {
		return models.G3MutationResponse{}, fmt.Errorf("could not create record: %w", err)
	}

	return created, nil
}

func (r *recordsClient) Update(record models.G3Record) (models.G3MutationResponse, error) {
	updated := models.G3MutationResponse{}
	err := r.transport.PutJSON(
		context.Background(),
		fmt.Sprintf("%s/%d", r.path, *record.Id),
		record,
		&updated,
	)

	if err != nil {
		return models.G3MutationResponse{}, fmt.Errorf("could not update record: %w", err)
	}

	return updated, nil
}

func (r *recordsClient) Delete(id string) error {
	return r.transport.Delete(context.Background(), fmt.Sprintf("%s/%s", r.path, id))
}
