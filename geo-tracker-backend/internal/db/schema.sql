CREATE TABLE IF NOT EXISTS prompt_sets (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS prompts (
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    prompt_set_id  BIGINT UNSIGNED,
    text           TEXT NOT NULL,
    category       VARCHAR(64) NOT NULL,
    active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    retired_at     DATETIME,
    notes          TEXT,
    FOREIGN KEY (prompt_set_id) REFERENCES prompt_sets(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS runs (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    prompt_set_id   BIGINT UNSIGNED,
    started_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at     DATETIME,
    duration_seconds INT NULL,
    prompt_count    INT NOT NULL DEFAULT 0,
    brand_count     INT NOT NULL DEFAULT 0,
    sample_count    INT NOT NULL DEFAULT 1,
    status          VARCHAR(32) NOT NULL DEFAULT 'running',
    total_cost_usd  DECIMAL(10,4),
    FOREIGN KEY (prompt_set_id) REFERENCES prompt_sets(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS results (
    id                    BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id                BIGINT UNSIGNED NOT NULL,
    prompt_id             BIGINT UNSIGNED NOT NULL,
    sample_index          TINYINT NOT NULL DEFAULT 0,
    provider              VARCHAR(32) NOT NULL,
    model_version         VARCHAR(128),
    brand                 VARCHAR(128) NOT NULL,
    raw_response          MEDIUMTEXT,
    brand_mentioned       BOOLEAN NOT NULL DEFAULT FALSE,
    sentiment             VARCHAR(32),
    mention_count         INT NOT NULL DEFAULT 0,
    recommendation_rank   INT,
    competitors_mentioned JSON,
    cited_urls            JSON,
    tokens_input          INT,
    tokens_output         INT,
    latency_ms            INT,
    cost_usd              DECIMAL(10,6),
    extraction_error      TEXT,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (run_id)    REFERENCES runs(id),
    FOREIGN KEY (prompt_id) REFERENCES prompts(id),
    INDEX idx_results_brand_provider (brand, provider),
    INDEX idx_results_run_id (run_id),
    INDEX idx_results_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS stability_scores (
    id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id            BIGINT UNSIGNED NOT NULL,
    prompt_id         BIGINT UNSIGNED NOT NULL,
    provider          VARCHAR(32) NOT NULL,
    brand             VARCHAR(128) NOT NULL,
    sample_count      INT NOT NULL,
    mention_rate      DECIMAL(5,2),
    rank_variance     DECIMAL(5,2),
    stability_score   DECIMAL(5,2),
    FOREIGN KEY (run_id)    REFERENCES runs(id),
    FOREIGN KEY (prompt_id) REFERENCES prompts(id),
    INDEX idx_stability_run (run_id, brand)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS visibility_scores (
    id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id            BIGINT UNSIGNED NOT NULL,
    brand             VARCHAR(255) NOT NULL,
    score             DECIMAL(6,2) NOT NULL DEFAULT 0,
    mention_rate      DECIMAL(6,2) NOT NULL DEFAULT 0,
    first_rec_rate    DECIMAL(6,2) NOT NULL DEFAULT 0,
    sentiment_score   DECIMAL(5,3) NOT NULL DEFAULT 0,
    citation_score    DECIMAL(6,2) NOT NULL DEFAULT 0,
    stability_score   DECIMAL(6,2) NOT NULL DEFAULT 0,
    provider_coverage DECIMAL(6,2) NOT NULL DEFAULT 0,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_brand_run (brand, run_id),
    FOREIGN KEY (run_id) REFERENCES runs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS explanations (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id      BIGINT UNSIGNED NOT NULL,
    brand       VARCHAR(255) NOT NULL,
    summary     TEXT NOT NULL,
    drivers     JSON NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_run_brand (run_id, brand)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS recommendations (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    run_id          BIGINT UNSIGNED NOT NULL,
    brand           VARCHAR(128) NOT NULL,
    priority        INT NOT NULL DEFAULT 1,
    category        VARCHAR(64),
    action          TEXT NOT NULL,
    expected_impact TEXT,
    rationale       TEXT,
    implemented_at  DATETIME,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (run_id) REFERENCES runs(id),
    INDEX idx_recommendations_brand (brand, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
