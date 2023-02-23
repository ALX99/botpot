*** Settings ***
Library             Process
Library             String
Library             OperatingSystem
Resource            keywords.robot

Test Teardown       Cleanup Database


*** Variables ***
${HONEYPOT}             localhost:22
${SSH_SERVER}           localhost:2001

${DB_CHECK_DELAY}       1s
@{DB_TABLES}            Session    Channel    Request    PTYRequest    ExecRequest    ExitStatusRequest    ExitSignalRequest    ShellRequest    WindowDimChangeRequest    EnvironmentRequest    SubSystemRequest


*** Test Cases ***
Basic tests
    [Documentation]    Verifies that output from a real SSH server is the same
    ...    as the output from the honeypot
    Verify same SSH output    ${HONEYPOT}    ${SSH_SERVER}    pwd
    Verify same SSH output    ${HONEYPOT}    ${SSH_SERVER}    whoami
    Verify same SSH output    ${HONEYPOT}    ${SSH_SERVER}    echo "$((1+1))"

IP address is inserted
    [Documentation]    Verifies that an entry in the IP table is created
    ...    after a SSH session has finished
    SSH    ${HONEYPOT}    ls
    Sleep    ${DB_CHECK_DELAY}
    ${res}=    Psql    SELECT COUNT(*) FROM ip;
    Should Be Equal As Numbers    1    ${res.stdout}

Cmd is inserted
    [Documentation]    Verifies that an entry in the ExecRequest table is
    ...    created after a SSH session has finished
    SSH    ${HONEYPOT}    ls
    Sleep    ${DB_CHECK_DELAY}
    ${res}=    Psql    SELECT COUNT(*) FROM ExecRequest;
    Should Be Equal As Numbers    1    ${res.stdout}
    ${res}=    Psql    SELECT command FROM ExecRequest;
    Should Be Equal As Strings    ls    ${res.stdout}


*** Keywords ***
Cleanup Database
    [Documentation]    Cleans up all the tables in the Database
    FOR    ${table}    IN    @{DB_TABLES}
        Psql    DELETE FROM ${table};
    END
