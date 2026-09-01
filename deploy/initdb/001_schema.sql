-- 多端租赁平台 —— 初始数据库结构（M1 核心表）
-- 由 docker-entrypoint-initdb.d 在 MySQL 首次启动时自动执行
-- 仅首次创建数据库时运行；后续表结构变更请使用迁移脚本

USE `rental`;

-- ---------- 用户表 ----------
CREATE TABLE IF NOT EXISTS `users` (
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `phone`         VARCHAR(20)     NOT NULL COMMENT '手机号（AES加密存储）',
    `phone_enc`     VARCHAR(255)    DEFAULT NULL COMMENT '手机号密文',
    `nickname`      VARCHAR(64)     DEFAULT '' COMMENT '昵称',
    `avatar`        VARCHAR(255)    DEFAULT '' COMMENT '头像URL',
    `real_name`     VARCHAR(64)     DEFAULT '' COMMENT '实名姓名（加密）',
    `id_card`       VARCHAR(255)    DEFAULT NULL COMMENT '身份证（AES加密）',
    `credit_score`  INT             NOT NULL DEFAULT 100 COMMENT '信用分 0-100',
    `deposit_bal`   DECIMAL(12,2)   NOT NULL DEFAULT 0.00 COMMENT '押金账户余额',
    `status`        TINYINT         NOT NULL DEFAULT 1 COMMENT '1正常 2禁用',
    `created_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_phone` (`phone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- ---------- 品类表 ----------
CREATE TABLE IF NOT EXISTS `categories` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`        VARCHAR(64)     NOT NULL COMMENT '品类名，如 3C数码/辅助器械/骑行设备',
    `parent_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父品类ID 0为顶级',
    `icon`        VARCHAR(255)    DEFAULT '' COMMENT '图标URL',
    `sort`        INT             NOT NULL DEFAULT 0 COMMENT '排序',
    `status`      TINYINT         NOT NULL DEFAULT 1 COMMENT '1启用 0停用',
    `created_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_parent` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='品类表';

-- ---------- 租赁物品表 ----------
CREATE TABLE IF NOT EXISTS `items` (
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `owner_id`      BIGINT UNSIGNED NOT NULL COMMENT '物品归属用户ID',
    `category_id`   BIGINT UNSIGNED NOT NULL COMMENT '品类ID',
    `title`         VARCHAR(128)    NOT NULL COMMENT '标题',
    `desc`          TEXT            COMMENT '描述',
    `images`        TEXT            COMMENT '图片JSON数组',
    `daily_price`   DECIMAL(12,2)   NOT NULL COMMENT '每日租金',
    `deposit`       DECIMAL(12,2)   NOT NULL DEFAULT 0.00 COMMENT '押金金额',
    `stock`         INT             NOT NULL DEFAULT 1 COMMENT '库存数量',
    `status`        TINYINT         NOT NULL DEFAULT 1 COMMENT '1上架 0下架 2已售罄',
    `city`          VARCHAR(64)     DEFAULT '' COMMENT '所在城市',
    `lat`           DECIMAL(10,7)   DEFAULT NULL COMMENT '纬度',
    `lng`           DECIMAL(10,7)   DEFAULT NULL COMMENT '经度',
    `created_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_owner` (`owner_id`),
    KEY `idx_category` (`category_id`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='租赁物品表';

-- ---------- 订单表 ----------
CREATE TABLE IF NOT EXISTS `orders` (
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_no`      VARCHAR(64)     NOT NULL COMMENT '订单号',
    `item_id`       BIGINT UNSIGNED NOT NULL COMMENT '物品ID',
    `renter_id`     BIGINT UNSIGNED NOT NULL COMMENT '租客用户ID',
    `owner_id`      BIGINT UNSIGNED NOT NULL COMMENT '房东用户ID',
    `start_date`    DATE            NOT NULL COMMENT '租期开始',
    `end_date`      DATE            NOT NULL COMMENT '租期结束',
    `days`          INT             NOT NULL COMMENT '租赁天数',
    `rent_amount`   DECIMAL(12,2)   NOT NULL COMMENT '租金总额',
    `deposit`       DECIMAL(12,2)   NOT NULL COMMENT '押金',
    `status`        TINYINT         NOT NULL DEFAULT 0 COMMENT '状态机见注释',
    `pay_trade_no`  VARCHAR(64)     DEFAULT NULL COMMENT '支付流水号',
    `cancel_reason` VARCHAR(255)    DEFAULT NULL COMMENT '取消原因',
    `created_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_no` (`order_no`),
    KEY `idx_renter` (`renter_id`),
    KEY `idx_owner` (`owner_id`),
    KEY `idx_item` (`item_id`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单表';
-- 订单状态机: 0待支付 1待取(已付租金) 2租赁中(押金冻结) 3待归还 4已归还(结算) 5已取消 6违约(扣押金)

-- ---------- 押金流水表 ----------
CREATE TABLE IF NOT EXISTS `deposits` (
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_id`      BIGINT UNSIGNED NOT NULL COMMENT '订单ID',
    `user_id`       BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `amount`        DECIMAL(12,2)   NOT NULL COMMENT '金额',
    `type`          TINYINT         NOT NULL COMMENT '1冻结 2解冻 3扣款',
    `ref`           VARCHAR(64)     DEFAULT NULL COMMENT '关联流水号',
    `created_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_order` (`order_id`),
    KEY `idx_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='押金流水表';

-- ---------- 支付流水表 ----------
CREATE TABLE IF NOT EXISTS `payments` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_id`        BIGINT UNSIGNED NOT NULL COMMENT '订单ID',
    `out_trade_no`    VARCHAR(64)     NOT NULL COMMENT '商户订单号',
    `transaction_id`  VARCHAR(64)     DEFAULT NULL COMMENT '微信支付单号',
    `channel`         VARCHAR(16)     NOT NULL DEFAULT 'wechat' COMMENT '支付渠道',
    `amount`          DECIMAL(12,2)   NOT NULL COMMENT '支付金额',
    `status`          TINYINT         NOT NULL DEFAULT 0 COMMENT '0待支付 1成功 2失败 3已退款',
    `raw_callback`    TEXT            COMMENT '微信回调原始报文',
    `created_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_out_trade_no` (`out_trade_no`),
    KEY `idx_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付流水表';

-- ---------- 初始品类数据 ----------
INSERT IGNORE INTO `categories` (`id`, `name`, `parent_id`, `sort`) VALUES
    (1, '3C数码', 0, 1),
    (2, '辅助器械', 0, 2),
    (3, '骑行设备', 0, 3),
    (4, '手机', 1, 1),
    (5, '相机', 1, 2),
    (6, '笔记本', 1, 3),
    (7, '轮椅', 2, 1),
    (8, '康复器械', 2, 2),
    (9, '自行车', 3, 1),
    (10, '骑行头盔', 3, 2);
