INSERT INTO pickup_points
    (external_id, type, name, address, city, region, postal_code, country, latitude, longitude, work_hours)
VALUES
    ('MSK-001', 'locker', 'Arbat Locker',
     '24 Arbat Street', 'Moscow', 'Moscow', '119002', 'RU',
     55.751244, 37.593409,
     '{"mon_fri":"09:00-21:00","sat_sun":"10:00-20:00"}'::jsonb),

    ('MSK-002', 'office', 'Tverskaya Pickup Office',
     '15 Tverskaya Street', 'Moscow', 'Moscow', '125009', 'RU',
     55.760673, 37.607670,
     '{"mon_fri":"10:00-20:00","sat":"11:00-18:00","sun":"closed"}'::jsonb),

    ('SPB-001', 'locker', 'Nevsky Locker',
     '100 Nevsky Prospect', 'Saint Petersburg', 'Leningrad Oblast', '191025', 'RU',
     59.932011, 30.360905,
     '{"mon_sun":"08:00-23:00"}'::jsonb),

    ('SPB-002', 'terminal', 'Moskovsky Terminal',
     '150 Moskovsky Prospect', 'Saint Petersburg', 'Leningrad Oblast', '196105', 'RU',
     59.864472, 30.320312,
     '{"mon_fri":"08:00-22:00","sat_sun":"09:00-21:00"}'::jsonb),

    ('KZN-001', 'locker', 'Bauman Locker',
     '44 Bauman Street', 'Kazan', 'Tatarstan', '420111', 'RU',
     55.789856, 49.121506,
     '{"mon_sun":"09:00-22:00"}'::jsonb);
