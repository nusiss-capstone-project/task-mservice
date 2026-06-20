-- Task Center schema initialization

CREATE TABLE IF NOT EXISTS task_group (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'DRAFT',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_group_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'DRAFT',
    condition_expressions varchar(255) NOT NULL DEFAULT '',
    start_time DATETIME NULL,
    end_time DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_task_group_id (task_group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_condition (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id BIGINT NOT NULL,
    no INT NOT NULL,
    data_metric_id BIGINT NOT NULL,
    data_operator_id BIGINT NOT NULL,
    condition_value VARCHAR(255) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_task_condition_task_id (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS data_metric (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(128) NOT NULL,
    data_source VARCHAR(255) NOT NULL,
    config JSON NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_data_metric_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS metric_operator (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    display VARCHAR(128) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_metric_operator_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_execution_progress (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'Init',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_task_user (task_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_condition_execution_progress (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    execution_id BIGINT NOT NULL,
    task_id BIGINT NOT NULL,
    condition_id BIGINT NOT NULL,
    current_value VARCHAR(255) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'Init',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_condition_progress_execution_id (execution_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO data_metric (code, data_source, config) VALUES
    ('net_deposit_volume', 'deposit.events', '{"field":"amount"}'),
    ('kyc_completed', 'user.events', '{"event":"kyc_completed"}')
ON DUPLICATE KEY UPDATE code = code;

INSERT INTO metric_operator (code, display) VALUES
    ('lt', 'is less than'),
    ('gt', 'is greater than'),
    ('eq', 'equals')
ON DUPLICATE KEY UPDATE code = code;
