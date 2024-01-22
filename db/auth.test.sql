-- password: password12345
INSERT INTO auth.users (id, username, email, password_hash, created_at, last_login_at, last_login_ip) VALUES
    ('0b73f55e-bec8-44c1-a00d-645ad7319933', 'glut', 'glut@example.com', '$2a$10$jD1CD9T2Gjy.ziO/raY6XOettyIdp2il8oHwcszQY3uAWdCp95pq6', now() - INTERVAL '15 days', now() - INTERVAL '5 days', '0.0.0.0'),
    ('f2fb78ed-8e17-44d3-b46d-349a78bf7014', 'glut2', 'glut@example.com', '$2a$10$jD1CD9T2Gjy.ziO/raY6XOettyIdp2il8oHwcszQY3uAWdCp95pq6', now() - INTERVAL '30 days', null, null),
    ('141ce8e4-c0b2-4b8e-80a3-72c1237fd19a', 'glut3', 'glut2@example.com', '$2a$10$jD1CD9T2Gjy.ziO/raY6XOettyIdp2il8oHwcszQY3uAWdCp95pq6', now() - INTERVAL '60 days', null, null),
    ('41e68f67-db31-4a8f-a1e9-31d3fa732a32', 'glut4', 'GLUT4@example.com', '$2a$10$jD1CD9T2Gjy.ziO/raY6XOettyIdp2il8oHwcszQY3uAWdCp95pq6', now() - INTERVAL '120 days', null, null);

INSERT INTO auth.sessions (id, token, user_id, user_ip, session_number, created_at, expires_at) VALUES
    ('d68ff336-0ae0-447c-aa18-65dad1409b38', 'AOOrbNViX4BTpXhr3Ffcq1EAw5dhoHTF', '0b73f55e-bec8-44c1-a00d-645ad7319933', '0.0.0.0', 1, now(), now() + INTERVAL '30 days'),
    ('032bcc68-36bc-4915-b672-b79aeedcb7a8', 'L1eXfyrOZ5OMo8Dgom3FkbAZ50tUxEMM', '141ce8e4-c0b2-4b8e-80a3-72c1237fd19a', '0.0.0.0', 1, now() - INTERVAL '2 days', now() - INTERVAL '1 day');    

INSERT INTO auth.roles (id, name, description, created_at, created_by) VALUES
    ('db531eca-1a7a-4768-9652-994f719b567e', 'admin', 'For do admin things.', now(), '0b73f55e-bec8-44c1-a00d-645ad7319933'),
    ('0f5ac467-5941-4cc3-9352-dbb2ef3ea3e8', 'moderator', 'For do mod things.', now(), 'f2fb78ed-8e17-44d3-b46d-349a78bf7014');

INSERT INTO auth.bans (user_id, reason, description, banned_by, banned_at, unbanned_at) VALUES
    ('f2fb78ed-8e17-44d3-b46d-349a78bf7014', 'spam', null, '0b73f55e-bec8-44c1-a00d-645ad7319933', now(), now());

INSERT INTO auth.permissions (id, name, description, created_at, created_by) VALUES
    ('3eeea1d9-936f-409e-adc2-0559e6414136', 'users:read', 'Read user data.', now(), '0b73f55e-bec8-44c1-a00d-645ad7319933'),
    ('2aac1e03-d0d1-4499-b73c-b1b64354af2f', 'users:delete', 'Delete user data.', now(), '0b73f55e-bec8-44c1-a00d-645ad7319933');

INSERT INTO auth.role_permissions (id, role_id, permission_id, created_at, created_by) VALUES
    ('baeb9b61-70a5-4d22-8f36-578649102f08', 'db531eca-1a7a-4768-9652-994f719b567e', '3eeea1d9-936f-409e-adc2-0559e6414136', now(), '0b73f55e-bec8-44c1-a00d-645ad7319933'),
    ('a204ed75-75b9-47af-8b5a-bafa8e03703b', 'db531eca-1a7a-4768-9652-994f719b567e', '2aac1e03-d0d1-4499-b73c-b1b64354af2f', now(), '0b73f55e-bec8-44c1-a00d-645ad7319933'),
    ('9958b21a-510a-40f5-8227-d3c061881494', '0f5ac467-5941-4cc3-9352-dbb2ef3ea3e8', '3eeea1d9-936f-409e-adc2-0559e6414136', now(), '0b73f55e-bec8-44c1-a00d-645ad7319933');

INSERT INTO auth.user_roles (id, role_id, user_id, created_at, created_by) VALUES
    ('53328f2f-c671-426d-a452-f7a97d066a2e', 'db531eca-1a7a-4768-9652-994f719b567e', '0b73f55e-bec8-44c1-a00d-645ad7319933', now(), null),
    ('d413dc99-7689-431b-9feb-631fe234894b', '0f5ac467-5941-4cc3-9352-dbb2ef3ea3e8', 'f2fb78ed-8e17-44d3-b46d-349a78bf7014', now(), '0b73f55e-bec8-44c1-a00d-645ad7319933');
