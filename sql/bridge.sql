create database bridge;

create user bridge login password 'bridge';

grant all on database bridge to bridge;

\c bridge bridge;

create table bridge ( id varchar(64) not null, ipaddress varchar(20), createdt timestamp, name varchar(30) not null);

