alter table ads add constraint ads_price_check check (price >= 0);
alter table ads add constraint categories_length_check check (array_length(categories, 1) between 1 and 5);