package service

import (
	"context"
	"sync"

	pb "kgs-platform/api/ontology/v1"
)

type OntologyService struct {
	pb.UnimplementedOntologyServer
	mu             sync.RWMutex
	byApp          map[string]*ontologyStore
	nextEntityID   uint32
	nextRelationID uint32
}

type ontologyStore struct {
	entities      map[string]*entityTypeItem
	entityOrder   []string
	relations     map[string]*relationTypeItem
	relationOrder []string
}

type entityTypeItem struct {
	ID     uint32
	Name   string
	Schema string
}

type relationTypeItem struct {
	ID               uint32
	Name             string
	PropertiesSchema string
	SourceTypes      []string
	TargetTypes      []string
}

func NewOntologyService() *OntologyService {
	return &OntologyService{
		byApp: make(map[string]*ontologyStore),
	}
}

func (s *OntologyService) CreateEntityType(ctx context.Context, req *pb.CreateEntityTypeRequest) (*pb.CreateEntityTypeReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetName() == "" {
		return &pb.CreateEntityTypeReply{Status: "INVALID"}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	store := s.ensureStoreLocked(appCtx.AppID)
	if item, ok := store.entities[req.GetName()]; ok {
		return &pb.CreateEntityTypeReply{
			Id:     item.ID,
			Name:   item.Name,
			Status: "EXISTS",
		}, nil
	}

	s.nextEntityID++
	item := &entityTypeItem{
		ID:     s.nextEntityID,
		Name:   req.GetName(),
		Schema: req.GetSchema(),
	}
	store.entities[item.Name] = item
	store.entityOrder = append(store.entityOrder, item.Name)

	return &pb.CreateEntityTypeReply{
		Id:     item.ID,
		Name:   item.Name,
		Status: "CREATED",
	}, nil
}
func (s *OntologyService) CreateRelationType(ctx context.Context, req *pb.CreateRelationTypeRequest) (*pb.CreateRelationTypeReply, error) {
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetName() == "" {
		return &pb.CreateRelationTypeReply{Status: "INVALID"}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	store := s.ensureStoreLocked(appCtx.AppID)
	if item, ok := store.relations[req.GetName()]; ok {
		return &pb.CreateRelationTypeReply{
			Id:     item.ID,
			Name:   item.Name,
			Status: "EXISTS",
		}, nil
	}

	s.nextRelationID++
	item := &relationTypeItem{
		ID:               s.nextRelationID,
		Name:             req.GetName(),
		PropertiesSchema: req.GetPropertiesSchema(),
		SourceTypes:      append([]string(nil), req.GetSourceTypes()...),
		TargetTypes:      append([]string(nil), req.GetTargetTypes()...),
	}
	store.relations[item.Name] = item
	store.relationOrder = append(store.relationOrder, item.Name)

	return &pb.CreateRelationTypeReply{
		Id:     item.ID,
		Name:   item.Name,
		Status: "CREATED",
	}, nil
}
func (s *OntologyService) ListEntityTypes(ctx context.Context, req *pb.ListEntityTypesRequest) (*pb.ListEntityTypesReply, error) {
	_ = req
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	store, ok := s.byApp[appCtx.AppID]
	if !ok {
		return &pb.ListEntityTypesReply{Entities: []*pb.EntityTypeInfo{}}, nil
	}

	out := make([]*pb.EntityTypeInfo, 0, len(store.entityOrder))
	for _, name := range store.entityOrder {
		item := store.entities[name]
		if item == nil {
			continue
		}
		out = append(out, &pb.EntityTypeInfo{
			Id:     item.ID,
			Name:   item.Name,
			Schema: item.Schema,
		})
	}
	return &pb.ListEntityTypesReply{Entities: out}, nil
}
func (s *OntologyService) ListRelationTypes(ctx context.Context, req *pb.ListRelationTypesRequest) (*pb.ListRelationTypesReply, error) {
	_ = req
	appCtx, err := getAppContext(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	store, ok := s.byApp[appCtx.AppID]
	if !ok {
		return &pb.ListRelationTypesReply{Relations: []*pb.RelationTypeInfo{}}, nil
	}

	out := make([]*pb.RelationTypeInfo, 0, len(store.relationOrder))
	for _, name := range store.relationOrder {
		item := store.relations[name]
		if item == nil {
			continue
		}
		out = append(out, &pb.RelationTypeInfo{
			Id:               item.ID,
			Name:             item.Name,
			PropertiesSchema: item.PropertiesSchema,
			SourceTypes:      append([]string(nil), item.SourceTypes...),
			TargetTypes:      append([]string(nil), item.TargetTypes...),
		})
	}
	return &pb.ListRelationTypesReply{Relations: out}, nil
}

func (s *OntologyService) ensureStoreLocked(appID string) *ontologyStore {
	if store, ok := s.byApp[appID]; ok {
		return store
	}
	store := &ontologyStore{
		entities:  make(map[string]*entityTypeItem),
		relations: make(map[string]*relationTypeItem),
	}
	s.byApp[appID] = store
	return store
}
