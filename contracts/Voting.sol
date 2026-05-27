// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract Voting {
    // Администратор задается один раз при деплое контракта. В нашем проекте
    // это локальный Ethereum-аккаунт admin из backend/config.json.
    address public admin;

    // Флаг управляет приемом голосов. Пока isActive == false, функция vote
    // будет отклонять транзакции через require.
    bool public isActive;

    // Кандидаты нумеруются с 1. Значение 0 считается несуществующим id.
    uint256 public candidatesCount;

    struct Candidate {
        // id нужен для удобного отображения во frontend и API.
        uint256 id;

        // name - публичное имя кандидата, например "Иванов".
        string name;

        // voteCount хранит количество голосов. Оно увеличивается только
        // функцией vote после прохождения всех проверок.
        uint256 voteCount;
    }

    mapping(uint256 => Candidate) public candidates;

    // Главная защита от повторного голосования: каждый Ethereum-адрес может
    // быть отмечен как проголосовавший только один раз.
    mapping(address => bool) public hasVoted;

    // Events не меняют состояние сами по себе. Они попадают в receipt logs,
    // чтобы backend/frontend могли наглядно показать, что произошло в блокчейне.
    // indexed-поля можно быстро фильтровать по topics.
    event CandidateAdded(uint256 indexed id, string name);
    event VotingStarted();
    event VotingStopped();
    event VoteCast(address indexed voter, uint256 indexed candidateId);

    modifier onlyAdmin() {
        // msg.sender - адрес, который подписал текущую транзакцию.
        // Для admin-действий это должен быть адрес, сохранившийся в constructor.
        require(msg.sender == admin, "Only admin");
        _;
    }

    constructor() {
        // Constructor выполняется только один раз: во время deploy-транзакции.
        // msg.sender здесь - аккаунт, который задеплоил контракт. В нашем
        // backend это adminPrivateKey, поэтому admin становится администратором.
        admin = msg.sender;
        isActive = false;
    }

    function addCandidate(string memory name) external onlyAdmin {
        // memory означает, что строка name существует только во время вызова.
        // После записи в mapping она сохраняется уже в storage контракта.
        require(bytes(name).length > 0, "Empty candidate name");

        // candidatesCount увеличивается перед записью, поэтому первый кандидат
        // получает id = 1. Это удобно для демонстрации и frontend select.
        candidatesCount++;
        candidates[candidatesCount] = Candidate(candidatesCount, name, 0);

        emit CandidateAdded(candidatesCount, name);
    }

    function startVoting() external onlyAdmin {
        // Запускать голосование может только admin. Также запрещаем запуск
        // без кандидатов, чтобы пользователь не голосовал в пустом списке.
        require(!isActive, "Voting already active");
        require(candidatesCount > 0, "No candidates");

        isActive = true;

        emit VotingStarted();
    }

    function stopVoting() external onlyAdmin {
        // Остановка нужна, чтобы после завершения голосования новые vote
        // транзакции больше не принимались.
        require(isActive, "Voting is not active");

        isActive = false;

        emit VotingStopped();
    }

    function vote(uint256 candidateId) external {
        // Эти проверки выполняются внутри блокчейна, поэтому их нельзя
        // обойти через прямой RPC-вызов или подмененный frontend.
        require(isActive, "Voting is not active");
        require(!hasVoted[msg.sender], "Already voted");
        require(candidateId > 0 && candidateId <= candidatesCount, "Candidate does not exist");

        // Сначала помечаем адрес как проголосовавший, затем увеличиваем счетчик.
        hasVoted[msg.sender] = true;
        candidates[candidateId].voteCount++;

        emit VoteCast(msg.sender, candidateId);
    }

    function getCandidate(uint256 candidateId) external view returns (uint256 id, string memory name, uint256 voteCount) {
        // view-функции не меняют состояние. Backend вызывает их через eth_call,
        // поэтому они не создают транзакцию и не попадают в блоки.
        require(candidateId > 0 && candidateId <= candidatesCount, "Candidate does not exist");

        Candidate memory candidate = candidates[candidateId];
        return (candidate.id, candidate.name, candidate.voteCount);
    }

    function getCandidatesCount() external view returns (uint256) {
        // Нужна backend-у, чтобы понять, сколько кандидатов прочитать.
        return candidatesCount;
    }

    function hasAddressVoted(address voter) external view returns (bool) {
        // Используется backend-ом и frontend-ом для проверки статуса voter.
        return hasVoted[voter];
    }
}
