INSERT INTO categories (slug, name_en, name_ru, name_es, sort_order) VALUES
    ('electronics', 'Electronics', 'Электроника', 'Electrónica', 10),
    ('clothing',    'Clothing',    'Одежда',      'Ropa',        20),
    ('home',        'Home & Garden','Дом и сад',  'Hogar y jardín', 30),
    ('books',       'Books',       'Книги',       'Libros',      40),
    ('sports',      'Sports',      'Спорт',       'Deportes',    50);

-- Subcategories
INSERT INTO categories (parent_id, slug, name_en, name_ru, name_es, sort_order)
SELECT id, 'smartphones', 'Smartphones', 'Смартфоны', 'Teléfonos', 10
FROM categories WHERE slug = 'electronics';

INSERT INTO categories (parent_id, slug, name_en, name_ru, name_es, sort_order)
SELECT id, 'laptops', 'Laptops', 'Ноутбуки', 'Portátiles', 20
FROM categories WHERE slug = 'electronics';

INSERT INTO categories (parent_id, slug, name_en, name_ru, name_es, sort_order)
SELECT id, 'mens-clothing', 'Men''s Clothing', 'Мужская одежда', 'Ropa de hombre', 10
FROM categories WHERE slug = 'clothing';

INSERT INTO categories (parent_id, slug, name_en, name_ru, name_es, sort_order)
SELECT id, 'womens-clothing', 'Women''s Clothing', 'Женская одежда', 'Ropa de mujer', 20
FROM categories WHERE slug = 'clothing';
