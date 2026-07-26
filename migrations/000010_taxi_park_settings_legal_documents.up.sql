CREATE TABLE taxi_park_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taxi_park_id UUID NOT NULL UNIQUE REFERENCES taxi_parks(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    short_name TEXT,
    support_phone TEXT,
    support_email TEXT,
    legal_name TEXT,
    legal_address TEXT,
    inn TEXT,
    ogrn TEXT,
    website TEXT,
    logo_url TEXT,
    primary_color VARCHAR(16),
    secondary_color VARCHAR(16),
    commission_percent NUMERIC(5,2),
    minimum_order_price_cents BIGINT NOT NULL DEFAULT 0,
    cancellation_timeout_sec INTEGER NOT NULL DEFAULT 300,
    driver_arrival_timeout_sec INTEGER NOT NULL DEFAULT 900,
    allow_cash_payment BOOLEAN NOT NULL DEFAULT true,
    allow_card_payment BOOLEAN NOT NULL DEFAULT true,
    allow_transfer_payment BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT taxi_park_settings_commission_range CHECK (
        commission_percent IS NULL OR (commission_percent >= 0 AND commission_percent <= 100)
    ),
    CONSTRAINT taxi_park_settings_minimum_price_non_negative CHECK (minimum_order_price_cents >= 0),
    CONSTRAINT taxi_park_settings_timeouts_positive CHECK (
        cancellation_timeout_sec > 0 AND driver_arrival_timeout_sec > 0
    )
);

CREATE TRIGGER taxi_park_settings_set_updated_at
BEFORE UPDATE ON taxi_park_settings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE taxi_park_tariffs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    base_price_cents BIGINT NOT NULL DEFAULT 0,
    price_per_km_cents BIGINT NOT NULL DEFAULT 0,
    price_per_minute_cents BIGINT NOT NULL DEFAULT 0,
    minimum_price_cents BIGINT NOT NULL DEFAULT 0,
    fixed_routes JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT taxi_park_tariffs_unique_name UNIQUE (taxi_park_id, name),
    CONSTRAINT taxi_park_tariffs_prices_non_negative CHECK (
        base_price_cents >= 0
        AND price_per_km_cents >= 0
        AND price_per_minute_cents >= 0
        AND minimum_price_cents >= 0
    )
);

CREATE TRIGGER taxi_park_tariffs_set_updated_at
BEFORE UPDATE ON taxi_park_tariffs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE legal_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_type TEXT NOT NULL,
    version VARCHAR(50) NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    language VARCHAR(8) NOT NULL DEFAULT 'ru',
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT legal_documents_type_check CHECK (
        document_type IN (
            'privacy_policy',
            'terms_of_service',
            'driver_agreement',
            'taxi_park_agreement',
            'consent_personal_data'
        )
    ),
    CONSTRAINT legal_documents_not_blank CHECK (
        length(trim(version)) > 0
        AND length(trim(title)) > 0
        AND length(trim(content)) > 0
        AND length(trim(language)) > 0
    )
);

CREATE UNIQUE INDEX idx_legal_documents_type_version_language
ON legal_documents (document_type, version, language);

CREATE UNIQUE INDEX idx_legal_documents_one_active
ON legal_documents (document_type, language)
WHERE is_active = true;

CREATE INDEX idx_legal_documents_type_created_at
ON legal_documents (document_type, created_at DESC);

CREATE TABLE user_document_acceptance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    document_id UUID NOT NULL REFERENCES legal_documents(id) ON DELETE RESTRICT,
    document_version VARCHAR(50) NOT NULL,
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip VARCHAR(64),
    user_agent TEXT,
    CONSTRAINT user_document_acceptance_unique UNIQUE (user_id, document_id)
);

CREATE INDEX idx_user_document_acceptance_user_accepted_at
ON user_document_acceptance (user_id, accepted_at DESC);

INSERT INTO legal_documents (document_type, version, title, content, language, is_active)
VALUES
(
    'terms_of_service',
    '1.0',
    'Пользовательское соглашение',
    '# Пользовательское соглашение\n\n## 1. Описание сервиса\nПлатформа предоставляет цифровой сервис для заказа такси, распределения заказов между водителями и таксопарками, realtime-уведомлений, геолокации и обработки обращений пользователей.\n\n## 2. Правила использования\nПользователь обязуется указывать достоверные данные, не нарушать работу сервиса, не создавать ложные заказы и соблюдать требования законодательства РФ.\n\n## 3. Правила заказа такси\nПользователь указывает точку подачи, точку назначения, тариф и способ оплаты. Итоговая стоимость может отличаться от предварительной оценки при изменении маршрута или условий поездки.\n\n## 4. Оплата\nОплата может производиться наличными, банковской картой или иным доступным способом. Доступность способов оплаты определяется настройками сервиса или таксопарка.\n\n## 5. Геолокация\nДля подбора водителя, расчета маршрута и отображения поездки сервис может использовать геолокацию пользователя и водителя.\n\n## 6. Ответственность сторон\nПлатформа отвечает за работоспособность цифровой инфраструктуры в пределах применимого законодательства. Водитель и/или таксопарк отвечают за фактическое оказание услуги перевозки.\n\n## 7. Ограничение ответственности\nПлатформа не несет ответственность за невозможность использования сервиса из-за отсутствия связи, ошибок устройства пользователя или действий третьих лиц.\n\n## 8. Блокировка аккаунта\nАккаунт может быть ограничен или заблокирован при нарушении правил, мошенничестве, угрозах безопасности или злоупотреблении сервисом.\n\n## 9. Контакты\nАктуальные контакты оператора и поддержки указываются в приложении или на сайте сервиса.\n',
    'ru',
    true
),
(
    'privacy_policy',
    '1.0',
    'Политика обработки персональных данных',
    '# Политика обработки персональных данных\n\n## 1. Перечень данных\nСервис может обрабатывать номер телефона, email, имя, данные профиля, геолокацию, сведения о заказах, платежные признаки, идентификаторы устройства и технические логи.\n\n## 2. Цели обработки\nДанные используются для регистрации, авторизации, выполнения заказов такси, связи с пользователем, обеспечения безопасности, расчетов, поддержки и улучшения качества сервиса.\n\n## 3. Сроки хранения\nДанные хранятся в течение срока действия аккаунта и сроков, необходимых для исполнения закона, бухгалтерского учета, урегулирования споров и защиты прав оператора.\n\n## 4. Передача третьим лицам\nДанные могут передаваться водителям, таксопаркам, платежным провайдерам, SMS/email-провайдерам, хостинг-поставщикам и государственным органам при наличии законных оснований.\n\n## 5. Права пользователя\nПользователь вправе получать сведения об обработке данных, требовать уточнения, блокирования или удаления данных в случаях, предусмотренных законом.\n\n## 6. Удаление данных\nЗапрос на удаление рассматривается оператором с учетом требований законодательства и необходимости хранения обязательных сведений.\n\n## 7. Cookies и идентификаторы устройств\nСервис может использовать cookies, push-токены и идентификаторы устройств для авторизации, аналитики, безопасности и доставки уведомлений.\n\n## 8. Геолокация\nГеолокация используется для определения точки подачи, поиска ближайших водителей, построения маршрута и контроля выполнения поездки.\n\n## 9. Контакты оператора\nКонтакты оператора размещаются в приложении, на сайте сервиса или в настройках таксопарка.\n',
    'ru',
    true
),
(
    'consent_personal_data',
    '1.0',
    'Согласие на обработку персональных данных',
    '# Согласие на обработку персональных данных\n\nПользователь подтверждает согласие на обработку персональных данных в соответствии с Федеральным законом РФ №152-ФЗ. Согласие включает сбор, запись, систематизацию, хранение, уточнение, использование, передачу, обезличивание, блокирование и удаление данных для целей работы сервиса такси.\n\nПользователь подтверждает, что согласие является явным, добровольным и информированным. Согласие действует до его отзыва, если иное не требуется законодательством РФ.\n',
    'ru',
    true
),
(
    'driver_agreement',
    '1.0',
    'Договор водителя',
    '# Договор водителя\n\n## 1. Правила работы\nВодитель обязуется соблюдать правила сервиса, поддерживать актуальный статус, своевременно обновлять геолокацию и корректно выполнять принятые заказы.\n\n## 2. Требования к водителю\nВодитель должен иметь необходимые документы, исправный автомобиль и соответствовать требованиям законодательства и таксопарка.\n\n## 3. Комиссия платформы\nС завершенных заказов удерживается комиссия платформы согласно действующим настройкам комиссии.\n\n## 4. Ответственность\nВодитель несет ответственность за качество и безопасность фактической перевозки, соблюдение ПДД и корректное взаимодействие с пассажирами.\n\n## 5. Блокировка\nДоступ водителя может быть ограничен при нарушении правил, жалобах, мошенничестве, угрозах безопасности или недостоверных документах.\n\n## 6. Геоданные\nВодитель соглашается на обработку геоданных во время работы на линии для распределения заказов, контроля поездки и безопасности.\n',
    'ru',
    true
),
(
    'taxi_park_agreement',
    '1.0',
    'Договор таксопарка',
    '# Договор таксопарка\n\n## 1. Условия подключения\nТаксопарк подключается к платформе для управления водителями, заказами, тарифами, расчетами и поддержкой клиентов.\n\n## 2. Комиссия\nКомиссия и финансовые условия определяются настройками платформы, индивидуальными соглашениями и действующими тарифами.\n\n## 3. Обязанности парка\nТаксопарк отвечает за проверку водителей, актуальность документов, качество перевозок, поддержку своих водителей и соблюдение законодательства.\n\n## 4. Ответственность\nТаксопарк несет ответственность за действия подключенных водителей в пределах применимого законодательства и договорных обязательств.\n\n## 5. Требования к водителям\nТаксопарк обязан допускать к работе только водителей, соответствующих требованиям законодательства и правилам сервиса.\n',
    'ru',
    true
);

