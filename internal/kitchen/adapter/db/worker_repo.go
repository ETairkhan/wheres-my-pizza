package db 

type WorkerRepo struct{
	ctx context.Context
	db core.IDB
}

func NewWorkerRepo(ctx context.Context, db core.IDB) core.IWorkerRepo{
	return &WorkerRepo{
		ctx: ctx,
		db: db,
	}
}

func (wr *)