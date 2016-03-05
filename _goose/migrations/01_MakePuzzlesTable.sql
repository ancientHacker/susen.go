
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
create table puzzles (
       sessionId text not null,
       puzzleId text not null,
       summary text,	 
       primary key (sessionId, puzzleId));
create index puzzles_puzzleId on puzzles (puzzleId);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
drop index puzzles_puzzleId;
drop table puzzles;


