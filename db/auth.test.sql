-- password: password12345
INSERT INTO auth.users (id, username, email, password_hash, created_at) VALUES
    ('0b73f55e-bec8-44c1-a00d-645ad7319933', 'glut', 'glut@example.com', '$2a$10$jD1CD9T2Gjy.ziO/raY6XOettyIdp2il8oHwcszQY3uAWdCp95pq6', CURRENT_TIMESTAMP),
    ('141ce8e4-c0b2-4b8e-80a3-72c1237fd19a', 'glut2', 'glut2@example.com', '$2a$10$jD1CD9T2Gjy.ziO/raY6XOettyIdp2il8oHwcszQY3uAWdCp95pq6', CURRENT_TIMESTAMP);

INSERT INTO auth.sessions (id, token, user_id, user_ip, created_at, expires_at) VALUES
    ('d68ff336-0ae0-447c-aa18-65dad1409b38', 'AOOrbNViX4BTpXhr3Ffcq1EAw5dhoHTF', '0b73f55e-bec8-44c1-a00d-645ad7319933', '0.0.0.0', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP + INTERVAL '30 days'),
    ('032bcc68-36bc-4915-b672-b79aeedcb7a8', 'L1eXfyrOZ5OMo8Dgom3FkbAZ50tUxEMM', '141ce8e4-c0b2-4b8e-80a3-72c1237fd19a', '0.0.0.0', CURRENT_TIMESTAMP - INTERVAL '2 days', CURRENT_TIMESTAMP - INTERVAL '1 day');    
