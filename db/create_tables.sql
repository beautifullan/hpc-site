CREATE TABLE software (id SERIAL PRIMARY KEY,name VARCHAR(200) NOT NULL UNIQUE,abstract TEXT,homepage TEXT,github TEXT,categories TEXT[],tags TEXT[],created_at TIMESTAMP DEFAULT NOW());
CREATE TABLE paper (id varchar(64) PRIMARY KEY,title TEXT NOT NULL,authors TEXT[],abstract TEXT,url TEXT,pdf TEXT,software_names TEXT[],created_at TIMESTAMP DEFAULT NOW());
CREATE TABLE benchmark (id SERIAL PRIMARY KEY,software_id INT NOT NULL,name TEXT,dataset_name TEXT,dataset_url TEXT,version TEXT,created_at TIMESTAMP DEFAULT NOW());
CREATE TABLE benchmark_results (id SERIAL PRIMARY KEY,benchmark_id INT NOT NULL,hardware JSONB,metrics JSONB,run_date TIMESTAMP DEFAULT NOW());
CREATE UNIQUE INDEX unique_software_idx ON software (name);