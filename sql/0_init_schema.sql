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
    player_one_win BOOLEAN,
    move_number INT,
    game_board TEXT,
    PRIMARY KEY (hosted_message_id),
    CONSTRAINT fk_game_user FOREIGN KEY (one_user_tg_id) REFERENCES user (tg_id),
    CONSTRAINT fk_game_user2 FOREIGN KEY (two_user_tg_id) REFERENCES user (tg_id)
);