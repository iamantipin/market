create index if not exists ads_title_idx on ads using gin (to_tsvector('simple', title));
create index if not exists ads_categories_idx on ads using gin (categories);