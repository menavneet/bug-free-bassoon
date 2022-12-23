-- Create the users table
CREATE TABLE users (
  id serial PRIMARY KEY,
  first_name text NOT NULL,
  last_name text NOT NULL,
  email text UNIQUE NOT NULL,
  password text NOT NULL
);

-- Insert a new user
INSERT INTO users (first_name, last_name, email, password)
VALUES ('John', 'Doe', 'john.doe@example.com', 'password');
