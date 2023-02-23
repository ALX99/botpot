*** Settings ***
Library             Process
Library             String
Library             OperatingSystem
Resource            keywords.robot

Test Teardown       Cleanup Database


*** Variables ***
${HONEYPOT}             localhost:22
${SSH_SERVER}           localhost:2001

${DB_CHECK_DELAY}       300ms
@{DB_TABLES}            Session    Channel    Request    PTYRequest    ExecRequest    ExitStatusRequest    ExitSignalRequest    ShellRequest    WindowDimChangeRequest    EnvironmentRequest    SubSystemRequest


*** Test Cases ***
Basic tests
    Verify same SSH output    ${HONEYPOT}    ${SSH_SERVER}    pwd
    Verify same SSH output    ${HONEYPOT}    ${SSH_SERVER}    whoami
    Verify same SSH output    ${HONEYPOT}    ${SSH_SERVER}    echo "$((1+1))"

IP address is inserted
    SSH    ${HONEYPOT}    ls
    Sleep    ${DB_CHECK_DELAY}
    ${res}=    Psql    SELECT COUNT(*) FROM ip;
    Should Be Equal As Numbers    1    ${res.stdout}


*** Keywords ***
Cleanup Database
    [Documentation]    Cleans up all the tables in the Database
    FOR    ${table}    IN    @{DB_TABLES}
        Psql    DELETE FROM ${table};
    END
