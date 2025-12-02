-- Go-IM 数据库初始化脚本
-- 该脚本在 MySQL 容器首次启动时自动执行

USE go_im;

-- 1. 消息表 (Timeline 的载体)
CREATE TABLE IF NOT EXISTS `timeline_message` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `msg_id` VARCHAR(64) NOT NULL,          -- 客户端生成的唯一ID，用于幂等去重
    `conversation_id` VARCHAR(64) NOT NULL, -- 会话ID，如 "group_101" 或 "private_u1_u2"
    `seq` BIGINT UNSIGNED NOT NULL,         -- 会话内序列号（核心字段）
    `sender_id` VARCHAR(64) NOT NULL,       -- 发送者ID
    `content` VARCHAR(4096),                -- 消息内容（限制长度，防止超大消息）
    `msg_type` TINYINT DEFAULT 1,           -- 1:文本, 2:图片
    `status` TINYINT DEFAULT 0,             -- 0:发送中, 1:已送达, 2:已读
    `send_time` BIGINT NOT NULL,            -- 发送时间戳
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE INDEX `uk_msg_id` (`msg_id`),                    -- 幂等去重索引
    UNIQUE INDEX `uk_conv_seq` (`conversation_id`, `seq`),  -- 核心：保证会话内seq唯一
    INDEX `idx_conv_seq` (`conversation_id`, `seq`)         -- 核心：用于范围拉取
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 2. 用户表
CREATE TABLE IF NOT EXISTS `user` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `user_id` VARCHAR(64) NOT NULL UNIQUE,
    `nickname` VARCHAR(64),
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 3. 用户会话状态表 (按会话维度存储ACK位点)
CREATE TABLE IF NOT EXISTS `user_conversation_state` (
    `user_id` VARCHAR(64) NOT NULL,
    `conversation_id` VARCHAR(64) NOT NULL,
    `last_ack_seq` BIGINT UNSIGNED DEFAULT 0,  -- 用户在该会话的最后确认序号
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`user_id`, `conversation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 4. 群成员表
CREATE TABLE IF NOT EXISTS `group_member` (
    `group_id` VARCHAR(64) NOT NULL,
    `user_id` VARCHAR(64) NOT NULL,
    `join_time` BIGINT NOT NULL,
    PRIMARY KEY (`group_id`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 插入测试数据
INSERT INTO `user` (`user_id`, `nickname`) VALUES
    ('user_1', '张三'),
    ('user_2', '李四'),
    ('user_3', '王五')
ON DUPLICATE KEY UPDATE `nickname` = VALUES(`nickname`);

-- 初始化一个测试群组
INSERT INTO `group_member` (`group_id`, `user_id`, `join_time`) VALUES
    ('group_1', 'user_1', UNIX_TIMESTAMP() * 1000),
    ('group_1', 'user_2', UNIX_TIMESTAMP() * 1000),
    ('group_1', 'user_3', UNIX_TIMESTAMP() * 1000)
ON DUPLICATE KEY UPDATE `join_time` = VALUES(`join_time`);
