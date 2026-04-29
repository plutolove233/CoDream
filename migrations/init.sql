-- CoDream Database Schema
-- PostgreSQL 16+
-- 创建时间: 2026-04-27

-- 启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- 1. Users 表 (用户)
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    password BYTEA NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_is_deleted ON users(is_deleted);
CREATE INDEX idx_users_created_at ON users(created_at DESC);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- ============================================
-- 2. Pipelines 表 (流水线定义)
-- ============================================
CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_by VARCHAR(255),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uk_pipelines_id_user UNIQUE (id, user_id)
);

CREATE INDEX idx_pipelines_user_id ON pipelines(user_id);
CREATE INDEX idx_pipelines_name ON pipelines(name);
CREATE INDEX idx_pipelines_status ON pipelines(status);
CREATE INDEX idx_pipelines_created_by ON pipelines(created_by);
CREATE INDEX idx_pipelines_is_deleted ON pipelines(is_deleted);
CREATE INDEX idx_pipelines_user_is_deleted ON pipelines(user_id, is_deleted);
CREATE INDEX idx_pipelines_created_at ON pipelines(created_at DESC);
CREATE INDEX idx_pipelines_deleted_at ON pipelines(deleted_at);

-- ============================================
-- 3. Pipeline Executions 表 (执行实例)
-- ============================================
CREATE TABLE IF NOT EXISTS pipeline_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    pipeline_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    current_stage_index INTEGER DEFAULT 0,
    input JSONB NOT NULL,
    output JSONB NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_pipeline_executions_pipeline_user
        FOREIGN KEY (pipeline_id, user_id) REFERENCES pipelines(id, user_id) ON DELETE CASCADE,
    CONSTRAINT uk_pipeline_executions_id_user UNIQUE (id, user_id)
);

CREATE INDEX idx_pipeline_executions_user_id ON pipeline_executions(user_id);
CREATE INDEX idx_pipeline_executions_pipeline_id ON pipeline_executions(pipeline_id);
CREATE INDEX idx_pipeline_executions_status ON pipeline_executions(status);
CREATE INDEX idx_pipeline_executions_is_deleted ON pipeline_executions(is_deleted);
CREATE INDEX idx_pipeline_executions_user_is_deleted ON pipeline_executions(user_id, is_deleted);
CREATE INDEX idx_pipeline_executions_created_at ON pipeline_executions(created_at DESC);
CREATE INDEX idx_pipeline_executions_deleted_at ON pipeline_executions(deleted_at);
CREATE INDEX idx_pipeline_executions_status_created ON pipeline_executions(status, created_at DESC);

-- ============================================
-- 4. Stage Executions 表 (阶段执行)
-- ============================================
CREATE TABLE IF NOT EXISTS stage_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    execution_id UUID NOT NULL,
    stage_name VARCHAR(255) NOT NULL,
    stage_order INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    input JSONB NOT NULL,
    output JSONB NOT NULL,
    plan JSONB NOT NULL,
    retry_count INTEGER DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_stage_executions_execution_user
        FOREIGN KEY (execution_id, user_id) REFERENCES pipeline_executions(id, user_id) ON DELETE CASCADE,
    CONSTRAINT uk_stage_executions_id_user UNIQUE (id, user_id)
);

CREATE INDEX idx_stage_executions_user_id ON stage_executions(user_id);
CREATE INDEX idx_stage_executions_execution_id ON stage_executions(execution_id);
CREATE INDEX idx_stage_executions_status ON stage_executions(status);
CREATE INDEX idx_stage_executions_is_deleted ON stage_executions(is_deleted);
CREATE INDEX idx_stage_executions_user_is_deleted ON stage_executions(user_id, is_deleted);
CREATE INDEX idx_stage_executions_created_at ON stage_executions(created_at DESC);
CREATE INDEX idx_stage_executions_deleted_at ON stage_executions(deleted_at);
CREATE INDEX idx_stage_executions_execution_status ON stage_executions(execution_id, status);

-- ============================================
-- 5. Checkpoints 表 (检查点)
-- ============================================
CREATE TABLE IF NOT EXISTS checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    execution_id UUID NOT NULL,
    stage_id UUID NOT NULL,
    position VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    artifacts JSONB NOT NULL,
    decision JSONB NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_checkpoints_execution_user
        FOREIGN KEY (execution_id, user_id) REFERENCES pipeline_executions(id, user_id) ON DELETE CASCADE,
    CONSTRAINT fk_checkpoints_stage_user
        FOREIGN KEY (stage_id, user_id) REFERENCES stage_executions(id, user_id) ON DELETE CASCADE
);

CREATE INDEX idx_checkpoints_user_id ON checkpoints(user_id);
CREATE INDEX idx_checkpoints_execution_id ON checkpoints(execution_id);
CREATE INDEX idx_checkpoints_stage_id ON checkpoints(stage_id);
CREATE INDEX idx_checkpoints_status ON checkpoints(status);
CREATE INDEX idx_checkpoints_is_deleted ON checkpoints(is_deleted);
CREATE INDEX idx_checkpoints_user_is_deleted ON checkpoints(user_id, is_deleted);
CREATE INDEX idx_checkpoints_deleted_at ON checkpoints(deleted_at);
CREATE INDEX idx_checkpoints_execution_status ON checkpoints(execution_id, status);

-- ============================================
-- 6. Agent Tasks 表 (Agent任务)
-- ============================================
CREATE TABLE IF NOT EXISTS agent_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    stage_execution_id UUID NOT NULL,
    agent_type VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    input JSONB NOT NULL,
    output JSONB NOT NULL,
    model_config JSONB NOT NULL,
    token_usage JSONB NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_agent_tasks_stage_user
        FOREIGN KEY (stage_execution_id, user_id) REFERENCES stage_executions(id, user_id) ON DELETE CASCADE
);

CREATE INDEX idx_agent_tasks_user_id ON agent_tasks(user_id);
CREATE INDEX idx_agent_tasks_stage_execution_id ON agent_tasks(stage_execution_id);
CREATE INDEX idx_agent_tasks_status ON agent_tasks(status);
CREATE INDEX idx_agent_tasks_agent_type ON agent_tasks(agent_type);
CREATE INDEX idx_agent_tasks_is_deleted ON agent_tasks(is_deleted);
CREATE INDEX idx_agent_tasks_user_is_deleted ON agent_tasks(user_id, is_deleted);
CREATE INDEX idx_agent_tasks_created_at ON agent_tasks(created_at DESC);
CREATE INDEX idx_agent_tasks_deleted_at ON agent_tasks(deleted_at);
CREATE INDEX idx_agent_tasks_stage_status ON agent_tasks(stage_execution_id, status);

-- ============================================
-- 触发器：自动更新 updated_at
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pipelines_updated_at BEFORE UPDATE ON pipelines
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pipeline_executions_updated_at BEFORE UPDATE ON pipeline_executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_stage_executions_updated_at BEFORE UPDATE ON stage_executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agent_tasks_updated_at BEFORE UPDATE ON agent_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
