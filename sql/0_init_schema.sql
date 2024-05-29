CREATE TABLE user (
    tg_id VARCHAR(45) NOT NULL,
    first_name VARCHAR(45) NOT NULL,
    date_created DATETIME,
    elo INT,
    PRIMARY KEY (tg_id)
);



CREATE TABLE game (
    hosted_message_id VARCHAR(45) NOT NULL,
    one_user_tg_id INT,
    two_user_tg_id INT,
    date_created DATETIME NOT NULL,
    date_completed DATETIME,
    move_number INT,
    game_board TEXT,
    last_move DATETIME,
    PRIMARY KEY (hosted_message_id),
    CONSTRAINT fk_game_user FOREIGN KEY (one_user_tg_id) REFERENCES user (tg_id),
    CONSTRAINT fk_game_user2 FOREIGN KEY (two_user_tg_id) REFERENCES user (tg_id)
);


CREATE TABLE game_outcome (
    hosted_message_id VARCHAR(45) NOT NULL,
    tg_id VARCHAR(45) NOT NULL,
    elo_adjustment INT NOT NULL,
    PRIMARY KEY (hosted_message_id, tg_id),
    CONSTRAINT fk_game_id FOREIGN KEY (hosted_message_id) REFERENCES game (hosted_message_id),
    CONSTRAINT fk_tg_id FOREIGN KEY (tg_id) REFERENCES user (tg_id)
);